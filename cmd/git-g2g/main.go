package main

import (
	"bufio"
	"context"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"g2g/pkg/pack"
	"g2g/pkg/specs"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
)

var logger = golog.Logger("git-server")

func main() {
	golog.SetAllLoggers(golog.LevelInfo)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize FS
	appDir := GetAppDir()
	if err := MkDir(appDir); err != nil {
		log.Fatalf("Failed to initialize application directory: %v", err)
	}
	repoDir := GetRepositoryDir()
	if err := MkDir(repoDir); err != nil {
		log.Fatalf("Failed to initialize repository directory: %v", err)
	}
	priv, err := loadPrivateKey()
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}

	// Initialize libp2p Host
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(specs.HostAddress),
		libp2p.Identity(priv),
	}
	node, err := libp2p.New(opts...)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	// Prints on STDOUT the libp2p addresses
	for _, addr := range node.Addrs() {
		hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", node.ID().Pretty()))
		p2pAddr := addr.Encapsulate(hostAddr).String()
		fmt.Printf("Serving on g2g://%s\n", p2pAddr)
	}

	// Associate stream protocols to git services
	node.SetStreamHandlerMatch(specs.UploadPackProto, func(i protocol.ID) bool { return strings.HasPrefix(string(i), specs.UploadPackProto) }, UploadPackHandler)
	node.SetStreamHandlerMatch(specs.ReceivePackProto, func(i protocol.ID) bool { return strings.HasPrefix(string(i), specs.ReceivePackProto) }, ReceivePackHandler)
	defer node.Close()

	// git-g2g terminates upon Ctrl-C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
}

func loadPrivateKey() (crypto.PrivKey, error) {
	keyPath := GetPrivKeyPath()
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		exec.Command("ssh-keygen", "-t", "ecdsa", "-q", "-f", keyPath, "-N", "", "-m", "PEM").Run()
	}
	blob, _ := os.ReadFile(keyPath)
	block, _ := pem.Decode(blob)
	if block == nil {
		return nil, fmt.Errorf("no PEM blob found")
	}
	return crypto.UnmarshalECDSAPrivateKey(block.Bytes)
}

func UploadPackHandler(s network.Stream) {
	defer s.Reset()

	dir := path.Base(string(s.Protocol()))
	cmd := exec.Command("git", "upload-pack", dir)
	cmd.Dir = GetRepositoryDir()
	stdin, _ := cmd.StdinPipe() // read fetch-pack, not used
	stdout, _ := cmd.StdoutPipe()

	go func() {
		scn := pack.NewScanner(stdout)
		for scn.Scan() {
			s.Write(scn.Bytes())
		}
	}()
	go func() {
		scn := pack.NewScanner(s)
		for scn.Scan() {
			stdin.Write(scn.Bytes())
		}
	}()

	if err := cmd.Start(); err != nil {
		logger.Warnln(err)
		return
	}

	if err := cmd.Wait(); err != nil {
		logger.Fatal(err)
	}
}

func ReceivePackHandler(s network.Stream) {
	defer s.Reset()

	dir := path.Base(string(s.Protocol()))
	cmd := exec.Command("git", "receive-pack", dir)
	cmd.Dir = GetRepositoryDir()
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()

	go func() {
		scn := pack.NewScanner(stdout)
		for scn.Scan() {
			s.Write(scn.Bytes())
		}
	}()
	go func() {
		scn := pack.NewScanner(s)
		for scn.Scan() {
			stdin.Write(scn.Bytes())
		}

		r := bufio.NewReader(s)
		b := make([]byte, 512)

		for {
			r.Read(b)
			stdin.Write(b)
		}
	}()

	if err := cmd.Start(); err != nil {
		logger.Warnln(err)
		return
	}

	if err := cmd.Wait(); err != nil {
		logger.Fatal(err)
	}
}
