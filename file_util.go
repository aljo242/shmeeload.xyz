package main

import (
	"fmt"
	"io"
	"os"
)

// Exists is a basic file util that says if a dir or file exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// CopyFile copies filename src to dst
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening file: %v : %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("error creating file: %v : %w", src, err)
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("error copying %v to %v : %w", src, dst, err)
	}

	return nil
}
