package main

import (
	"bufio"
	"context"
	"os"
	"sync"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

var logger = golog.Logger("remote-helper")

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	args := os.Args
	if len(args) < 3 {
		logger.Fatalln("Usage: git-remote-g2g <remoteName> <multiAddr>")
	}

	node, err := libp2p.New()
	if err != nil {
		logger.Fatalln(err)
	}
	defer node.Close()

	kdht, err := NewDHT(ctx, node)
	if err != nil {
		logger.Fatal(err)
	}

	// TODO - now the repo style is changed
	repo, err := NewRepository(kdht, args[2])
	if err != nil {
		logger.Fatalln(err)
	}
	repo.AddAddressInto(node)

	stdinReader := bufio.NewReader(os.Stdin)
	for {
		command, err := stdinReader.ReadString('\n')
		if err != nil {
			logger.Fatalln(err)
		}

		switch {
		case command == "capabilities\n":
			PrintCapabilities(os.Stdout)
		case command == "connect git-upload-pack\n":
			if err = ConnectUploadPack(node, ctx, repo.addrs[0].ID, repo.id); err != nil {
				logger.Fatalln(err)
			}
		case command == "connect git-receive-pack\n":
			if err = ConnectReceivePack(node, ctx, repo.addrs[0].ID, repo.id); err != nil {
				logger.Fatalln(err)
			}
		default:
			logger.Fatalf("Unknown command: %q", command)
		}
	}
}

func NewDHT(ctx context.Context, host host.Host) (kdht *dht.IpfsDHT, err error) {
	kdht, err = dht.New(ctx, host)
	if err != nil {
		return
	}

	if err = kdht.Bootstrap(ctx); err != nil {
		return
	}

	var wg sync.WaitGroup
	for _, peerAddr := range dht.DefaultBootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := host.Connect(ctx, *peerinfo); err != nil {
				logger.Errorf("Error while connecting to node %q: %-v", peerinfo, err)
			} else {
				logger.Debugf("Connection established with bootstrap node: %q", *peerinfo)
			}
		}()
	}
	wg.Wait()

	return
}
