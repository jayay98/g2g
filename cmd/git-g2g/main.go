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
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
)

var logger = golog.Logger("git-server")

func main() {
	golog.SetAllLoggers(golog.LevelInfo)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize FS
	appDir := getAppDir()
	if err := mkDir(appDir); err != nil {
		log.Fatalf("Failed to initialize application directory: %v", err)
	}
	repoDir := getRepositoryDir()
	if err := mkDir(repoDir); err != nil {
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
	defer node.Close()
	// ids, err := identify.NewIDService(node)
	// if err != nil {
	// 	log.Fatalf("Failed to initialize identity service")
	// }
	// defer ids.Close()
	// hps, err := holepunch.NewService(node, ids)
	// if err != nil {
	// 	log.Fatalf("Failed to initialize holepunch service")
	// }
	// defer hps.Close()

	// Boostrap
	var wg sync.WaitGroup
	ctx := context.Background()
	for _, peerAddr := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := node.Connect(ctx, *peerinfo); err != nil {
				logger.Warning(err)
			} else {
				logger.Debug("Connection established with bootstrap node:", *peerinfo)
				relayaddr, _ := ma.NewMultiaddr("/p2p/" + peerinfo.ID.String() + "/p2p-circuit/p2p/" + node.ID().String())
				_, err = client.Reserve(context.Background(), node, *peerinfo)
				if err != nil {
					log.Printf("server failed to receive a relay reservation from relays. %v", err)
					return
				}
				for _, addr := range peerinfo.Addrs {
					p2pAddr := addr.Encapsulate(relayaddr).String()
					fmt.Printf("Serving on g2g://%s\n", p2pAddr)
				}
			}
		}()
	}
	wg.Wait()

	// TODO - Relay
	// TODO - Upgrade to holepunching

	// // Prints on STDOUT the libp2p addresses
	for _, addr := range node.Addrs() {
		hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", node.ID().Pretty()))
		p2pAddr := addr.Encapsulate(hostAddr).String()
		fmt.Printf("Serving on g2g://%s\n", p2pAddr)
	}

	// Associate stream protocols to git services
	node.SetStreamHandlerMatch(specs.UploadPackProto, func(i protocol.ID) bool { return strings.HasPrefix(string(i), specs.UploadPackProto) }, uploadPackHandler)
	node.SetStreamHandlerMatch(specs.ReceivePackProto, func(i protocol.ID) bool { return strings.HasPrefix(string(i), specs.ReceivePackProto) }, receivePackHandler)

	// git-g2g terminates upon Ctrl-C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
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
