package main

import (
	"os"
	"path"
)

func mkDir(dir string) (err error) {
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
	}

	return
}

func getAppDir() string {
	home, _ := os.UserHomeDir()
	return path.Join(home, ".g2g")
}

func getRepositoryDir() string {
	appDir := getAppDir()
	return path.Join(appDir, "repos")
}

func getPrivKeyPath() string {
	appDir := getAppDir()
	return path.Join(appDir, "key.pem")
}
