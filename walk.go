package main

import (
	"os"
	"path/filepath"
)

func IsSymlink(info os.FileInfo) bool {
	return info.Mode()&os.ModeSymlink == os.ModeSymlink
}

func ReadSymlink(path string, info os.FileInfo) (string, os.FileInfo, error) {
	var err error
	for IsSymlink(info) {
		link, err := os.Readlink(path)
		if err != nil {
			return path, info, err
		}
		linkAbs := link
		if !filepath.IsAbs(link) {
			linkAbs = filepath.Join(filepath.Dir(path), link)
		}
		path = linkAbs
		info, err = os.Lstat(path)
		if err != nil {
			return path, info, err
		}
	}
	return path, info, err
}
