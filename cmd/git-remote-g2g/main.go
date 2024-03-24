package main

import (
	"bufio"
	"context"
	"os"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
)

var logger = golog.Logger("remote-helper")

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	args := os.Args
	if len(args) < 3 {
		logger.Fatalln("Usage: git-remote-g2g <remoteName> <multiAddr>")
	}

	repo, err := NewRepository(args[2])
	if err != nil {
		logger.Fatalln(err)
	}

	node, err := libp2p.New()
	if err != nil {
		logger.Fatalln(err)
	}
	defer node.Close()

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
			if err = ConnectUploadPack(node, ctx, repo.address.ID, repo.id); err != nil {
				logger.Fatalln(err)
			}
		case command == "connect git-receive-pack\n":
			if err = ConnectReceivePack(node, ctx, repo.address.ID, repo.id); err != nil {
				logger.Fatalln(err)
			}
		default:
			logger.Fatalf("Unknown command: %q", command)
		}
	}
}
