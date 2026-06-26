// Command genwebp precomputes lossless WebP variants of the PNG and GIF images
// under a site directory, writing "<name>.webp" next to each source whenever the
// WebP is smaller. It runs at build time so the server embeds the variants and
// serves them without encoding anything at startup.
package main

import (
	"bytes"
	"fmt"
	"image"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/HugoSmits86/nativewebp"

	_ "image/gif"
	_ "image/png"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: genwebp <site-dir>")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "genwebp:", err)
		os.Exit(1)
	}
}

func run(root string) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		switch strings.ToLower(filepath.Ext(p)) {
		case ".png", ".gif":
		default:
			return nil
		}
		dst := p + ".webp"
		if upToDate(dst, p) {
			return nil
		}
		return convert(p, dst)
	})
}

// upToDate reports whether dst exists and is at least as new as src, so an
// unchanged image is not re-encoded on every build.
func upToDate(dst, src string) bool {
	di, err := os.Stat(dst)
	if err != nil {
		return false
	}
	si, err := os.Stat(src)
	if err != nil {
		return false
	}
	return !di.ModTime().Before(si.ModTime())
}

func convert(src, dst string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		// Not a decodable image (e.g. a malformed file); leave it alone.
		fmt.Fprintf(os.Stderr, "genwebp: skip %s: %v\n", src, err)
		return nil
	}
	var buf bytes.Buffer
	if err := nativewebp.Encode(&buf, img, nil); err != nil {
		fmt.Fprintf(os.Stderr, "genwebp: skip %s: encode: %v\n", src, err)
		return nil
	}
	// Only keep the WebP when it actually beats the source; drop a stale one
	// that no longer helps so the server never embeds a larger variant.
	if buf.Len() >= len(raw) {
		_ = os.Remove(dst)
		return nil
	}
	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Printf("genwebp: %s (%d -> %d, %d%%)\n", src, len(raw), buf.Len(), buf.Len()*100/len(raw))
	return nil
}
