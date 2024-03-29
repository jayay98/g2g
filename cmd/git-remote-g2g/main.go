package main

import (
	"bufio"
	"context"
	"fmt"
	"g2g/pkg/pack"
	"g2g/pkg/specs"
	"io"
	"os"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
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

	repo, err := NewRepo(args[2])
	if err != nil {
		logger.Fatalln(err)
	}

	node, err := libp2p.New()
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
			if err = ConnectUploadPack(node, ctx, repo.address.ID); err != nil {
				logger.Fatalln(err)
			}
		case command == "connect git-receive-pack\n":
			if err = ConnectReceivePack(node, ctx, repo.address.ID); err != nil {
				logger.Fatalln(err)
			}
		default:
			logger.Fatalf("Unknown command: %q", command)
		}
	}
}

func PrintCapabilities(w io.Writer) {
	fmt.Fprintln(w, "connect")
	fmt.Fprintln(w, "")
}

func ConnectUploadPack(node host.Host, ctx context.Context, peerId peer.ID) (err error) {
	// Connects to given git service.
	s, err := node.NewStream(ctx, peerId, specs.UploadPackProto)
	if err != nil {
		return err
	}
	os.Stdout.WriteString("\n")

	// After line feed terminating the positive (empty) response, the output of service starts.
	// Server advertises refs
	serviceScanner := pack.NewScanner(s)
	for serviceScanner.Scan() {
		fmt.Fprint(os.Stdout, serviceScanner.Text())
		if serviceScanner.Text() == "0000" {
			break
		}
	}

	// Client states "want" and "have"
	cmdScanner := pack.NewScanner(os.Stdin)
	for cmdScanner.Scan() {
		s.Write(cmdScanner.Bytes())
		if cmdScanner.Text() == "0009done\n" {
			break
		}
	}

	// Server optionally supply packfile
	for serviceScanner.Scan() {
		fmt.Fprint(os.Stdout, serviceScanner.Text())
		if serviceScanner.Text() == "0000" {
			break
		}
	}

	// After the connection ends, the remote helper exits.
	s.Reset()
	os.Exit(0)
	return
}

func ConnectReceivePack(node host.Host, ctx context.Context, peerId peer.ID) (err error) {
	// Connects to given git service.
	s, err := node.NewStream(ctx, peerId, specs.ReceivePackProto)
	if err != nil {
		return err
	}
	os.Stdout.WriteString("\n")

	// After line feed terminating the positive (empty) response, the output of service starts.
	// Server advertises refs
	serviceScanner := pack.NewScanner(s)
	for serviceScanner.Scan() {
		fmt.Fprint(os.Stdout, serviceScanner.Text())
		if serviceScanner.Text() == "0000" {
			break
		}
	}

	// Client states "want" and "have"
	cmdScanner := pack.NewScanner(os.Stdin)
	for cmdScanner.Scan() {
		s.Write(cmdScanner.Bytes())
		if cmdScanner.Text() == "0000" {
			break
		}
	}

	go func() {
		r := bufio.NewReader(os.Stdin)
		b := make([]byte, 1024)

		for {
			_, err = r.Read(b)
			if err != nil {
				logger.Warnln(err)
			}
			logger.Infoln(b)
			s.Write(b)
		}
	}()

	// Server ack
	for serviceScanner.Scan() {
		logger.Debugln(serviceScanner.Text())
		fmt.Fprint(os.Stdout, serviceScanner.Text())
		if serviceScanner.Text() == "000eunpack ok" {
			break
		}
	}

	// After the connection ends, the remote helper exits.
	s.Reset()
	os.Exit(0)
	return
}
