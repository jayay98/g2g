package tests

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

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
	require.NoError(t, server.Start())

	go func() {
		addr := <-ch
		remoteAddr := addr + "/" + path.Base(dir)
		cloneDir := t.TempDir()
		require.NoError(t, exec.Command("git", "clone", remoteAddr, cloneDir).Run())

		os.WriteFile(path.Join(cloneDir, "README.md"), []byte("# Sample"), 0644)
		cmd := exec.Command("git", "add", ".")
		cmd.Dir = cloneDir
		require.NoError(t, cmd.Run())

		cmd = exec.Command("git", "commit", "-m", "First commit")
		cmd.Dir = cloneDir
		require.NoError(t, cmd.Run())

		cmd = exec.Command("git", "push")
		cmd.Dir = cloneDir
		require.NoError(t, cmd.Run())

		cmd = exec.Command("git", "--no-pager", "log", "--pretty=oneline")
		cmd.Dir = dir
		serverLog, _ := cmd.Output()

		cmd = exec.Command("git", "--no-pager", "log", "--pretty=oneline")
		cmd.Dir = cloneDir
		clientLog, _ := cmd.Output()

		require.Equal(t, serverLog, clientLog)
		cancel()
	}()

	server.Wait()
}
