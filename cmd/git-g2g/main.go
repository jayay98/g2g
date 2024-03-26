package main

import (
	"context"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"g2g/pkg/specs"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var logger = golog.Logger("git-server")

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize FS
	appDir := getAppDir()
	if err := mkDir(appDir); err != nil {
		logger.Fatalf("Failed to initialize application directory: %v", err)
	}
	repoDir := getRepositoryDir()
	if err := mkDir(repoDir); err != nil {
		logger.Fatalf("Failed to initialize repository directory: %v", err)
	}
	priv, err := loadPrivateKey()
	if err != nil {
		logger.Fatalf("Failed to load private key: %v", err)
	}

	// Initialize libp2p Host
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(specs.HostAddress),
		libp2p.Identity(priv),
	}
	node, err := libp2p.New(opts...)
	if err != nil {
		logger.Fatalf("Failed to parse private key: %v", err)
	}
	defer node.Close()
	log.Printf("Host ID: %s", node.ID().Pretty())

	kdht, err := dht.New(ctx, node)
	if err != nil {
		logger.Fatal(err)
	}

	if err = kdht.Bootstrap(ctx); err != nil {
		logger.Fatal(err)
	}

	var wg sync.WaitGroup
	for _, peerAddr := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := node.Connect(ctx, *peerinfo); err != nil {
				logger.Errorf("Error while connecting to node %q: %-v", peerinfo, err)
			} else {
				logger.Debugf("Connection established with bootstrap node: %q", *peerinfo)
			}
		}()
	}
	wg.Wait()

	// Associate stream protocols to git services
	node.SetStreamHandlerMatch(specs.UploadPackProto, func(i protocol.ID) bool { return strings.HasPrefix(string(i), specs.UploadPackProto) }, uploadPackHandler)
	node.SetStreamHandlerMatch(specs.ReceivePackProto, func(i protocol.ID) bool { return strings.HasPrefix(string(i), specs.ReceivePackProto) }, receivePackHandler)

	// git-g2g terminates upon Ctrl-C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
}

func loadPrivateKey() (crypto.PrivKey, error) {
	keyPath := getPrivKeyPath()
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
