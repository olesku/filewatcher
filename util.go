package main

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// GetChecksum Get MD5 checksum for the contents in a file.
func GetChecksum(filePath string) (string, error) {
	// Get md5checksum of whole file.
	hash := md5.New()

	// Open file for reading.
	fh, err := os.Open(filePath)
	if err != nil {
		return "", err
	}

	defer fh.Close()

	if _, err := io.Copy(hash, fh); err == nil {
		hashInBytes := hash.Sum(nil)[:16]
		return hex.EncodeToString(hashInBytes), nil
	}

	return "", err
}

// ExitIfError If err is not nil exit with the corresponding error message.
func ExitIfError(err error) {
	if err != nil {
		log.Fatalf("Error: %s\n", err.Error())
	}
}

// ListDirectories List all directories in path recursively.
func ListDirectories(path string) []string {
	var directoryList []string

	walkFunc := func(wPath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			directoryList = append(directoryList, wPath)
		}

		return nil
	}

	filepath.Walk(path, walkFunc)

	return directoryList
}

// ListFiles List all files in a directory and subdirectories.
func ListFiles(path string) []string {
	var fileList []string

	walkFunc := func(wPath string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			fileList = append(fileList, wPath)
		}

		return nil
	}

	filepath.Walk(path, walkFunc)

	return fileList
}

// IsDirectory Check if given path is a directory.
func IsDirectory(path string) bool {
	fInfo, err := os.Stat(path)

	if err != nil {
		return false
	}

	if fInfo.IsDir() {
		return true
	}

	return false
}

// StripBasepath from path.
func StripBasepath(path string) (string, error) {
	cwd, err := os.Getwd()

	if err != nil {
		return path, err
	}

	path = strings.TrimPrefix(path, cwd)
	path = strings.TrimPrefix(path, "/")

	return path, nil
}
