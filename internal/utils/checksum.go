package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func CalculateChecksum(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if !info.IsDir() {
		return calculateFileChecksum(path)
	}

	return calculateDirectoryChecksum(path)
}

func calculateFileChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()

	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func calculateDirectoryChecksum(root string) (string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Strings(files)

	hash := sha256.New()
	for _, path := range files {
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return "", err
		}

		if _, err := hash.Write([]byte(relPath)); err != nil {
			return "", err
		}
		if _, err := hash.Write([]byte{0}); err != nil {
			return "", err
		}

		file, err := os.Open(path)
		if err != nil {
			return "", err
		}

		if _, err := io.Copy(hash, file); err != nil {
			file.Close()
			return "", err
		}
		file.Close()

		if _, err := hash.Write([]byte{0}); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
