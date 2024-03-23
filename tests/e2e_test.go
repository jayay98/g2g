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
	home, _ := os.UserHomeDir()
	parentDir := path.Join(home, ".g2g", "repos")
	dir, _ := os.MkdirTemp(parentDir, "*.git")
	_, err := exec.Command("git", "init", "--bare", dir).CombinedOutput()
	return dir, err
}

func TestMain(t *testing.T) {
	dir, err := InitBareRepository(t)
	if err != nil {
		t.Error("Could not initiate random bare repository.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { os.RemoveAll(dir); cancel() })

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
		remoteAddr := addr + "/" + path.Base(dir)
		cloneDir := t.TempDir()
		if err := exec.Command("git", "clone", remoteAddr, cloneDir).Run(); err != nil {
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
		cmd.Dir = dir
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
