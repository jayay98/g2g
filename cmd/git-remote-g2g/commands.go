package main

import (
	"bufio"
	"context"
	"fmt"
	"g2g/pkg/pack"
	"g2g/pkg/specs"
	"io"
	"os"
	"strings"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

func PrintCapabilities(w io.Writer) {
	fmt.Fprintln(w, "connect")
	fmt.Fprintln(w, "")
}

func ConnectUploadPack(node host.Host, ctx context.Context, peerId peer.ID, repository string) (err error) {
	// Connects to given git service.
	proto := strings.Join([]string{specs.UploadPackProto, repository}, "/")
	transient := network.WithUseTransient(ctx, proto)
	s, err := node.NewStream(transient, peerId, protocol.ConvertFromStrings([]string{proto})...)
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

func ConnectReceivePack(node host.Host, ctx context.Context, peerId peer.ID, repository string) (err error) {
	// Connects to given git service.
	proto := strings.Join([]string{specs.ReceivePackProto, repository}, "/")
	transient := network.WithUseTransient(ctx, proto)
	s, err := node.NewStream(transient, peerId, protocol.ConvertFromStrings([]string{proto})...)
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
