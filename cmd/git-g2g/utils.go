package main

import (
	"os"
	"path"
)

func MkDir(dir string) (err error) {
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
	}

	return
}

func GetAppDir() string {
	home, _ := os.UserHomeDir()
	return path.Join(home, ".g2g")
}

func GetRepositoryDir() string {
	appDir := GetAppDir()
	return path.Join(appDir, "repos")
}

func GetPrivKeyPath() string {
	appDir := GetAppDir()
	return path.Join(appDir, "key.pem")
}
