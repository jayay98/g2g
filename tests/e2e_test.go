package tests

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
)

func InitBareRepository(t *testing.T) (string, error) {
	// dir := t.TempDir()
	dir := "/tmp/test_repo"
	_, err := exec.Command("git", "init", "--bare", dir).Output()
	return dir, err
}

func TestMain(t *testing.T) {
	dir, err := InitBareRepository(t)
	if err != nil {
		t.Error("Could not initiate random bare repository.")
	}
	defer os.RemoveAll(dir)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := exec.CommandContext(ctx, "git-g2g", "git", "g2g")
	serverOut, _ := server.StdoutPipe()
	scn := bufio.NewScanner(serverOut)

	ch := make(chan string)
	go func() {
		scn.Scan()
		ma := strings.TrimSpace(strings.TrimPrefix(scn.Text(), "Serving on "))
		ch <- ma
		close(ch)
	}()
	if err = server.Start(); err != nil {
		t.Error(err)
	}

	go func() {
		addr := <-ch
		cloneDir := t.TempDir()
		if err := exec.Command("git", "clone", addr, cloneDir).Run(); err != nil {
			t.Fail()
		}

		os.WriteFile(path.Join(cloneDir, "README.md"), []byte("# Sample"), 0644)
		cmd := exec.Command("git", "add", ".")
		cmd.Dir = cloneDir
		if err = cmd.Run(); err != nil {
			t.Error(err)
		}

		cmd = exec.Command("git", "commit", "-m", "First commit")
		cmd.Dir = cloneDir
		if err = cmd.Run(); err != nil {
			t.Error(err)
		}

		cmd = exec.Command("git", "push")
		cmd.Dir = cloneDir
		if err = cmd.Run(); err != nil {
			t.Error(err)
		}

		cmd = exec.Command("git", "--no-pager", "log", "--pretty=oneline")
		cmd.Dir = "/tmp/test_repo"
		serverLog, _ := cmd.Output()

		cmd = exec.Command("git", "--no-pager", "log", "--pretty=oneline")
		cmd.Dir = cloneDir
		clientLog, _ := cmd.Output()

		if !bytes.Equal(serverLog, clientLog) {
			t.Fail()
		}
		cancel()
	}()

	server.Wait()
}
