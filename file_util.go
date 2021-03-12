package main

import (
	"os"
	"fmt"
	"io"
)

// Exists is a basic file util that says if a dir or file exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		return true // path/file exists
	}
	return false // path/file does not exist
}

// CopyFile copies filename src to dst
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("Error opening file: %v : %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("Error creating file: %v : %w", src, err)
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("Error copying %v to %v : %w", src, dst, err)
	}

	return nil
}
