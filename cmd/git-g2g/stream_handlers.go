package main

import (
	"bufio"
	"g2g/pkg/pack"
	"os/exec"
	"path"

	"github.com/libp2p/go-libp2p/core/network"
)

func uploadPackHandler(s network.Stream) {
	defer s.Reset()

	dir := path.Base(string(s.Protocol()))
	cmd := exec.Command("git", "upload-pack", dir)
	cmd.Dir = getRepositoryDir()
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

func receivePackHandler(s network.Stream) {
	defer s.Reset()

	dir := path.Base(string(s.Protocol()))
	cmd := exec.Command("git", "receive-pack", dir)
	cmd.Dir = getRepositoryDir()
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
