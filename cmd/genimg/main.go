// Command genimg precomputes smaller image variants under a site directory: a
// lossless WebP for each PNG/GIF, and a lossy AVIF for each PNG/GIF/JPEG. Each
// is written as a "<name>.webp" / "<name>.avif" sibling, kept only when it beats
// the source. It runs at build time so the server embeds the variants and serves
// them without encoding anything at startup.
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
	"github.com/gen2brain/avif"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// avifOptions favors quality (q80, 4:4:4 chroma keeps pixel-art edges sharp)
// while still landing far below the WebP variant. Speed 8 keeps the build fast.
var avifOptions = avif.Options{
	Quality:           80,
	QualityAlpha:      80,
	Speed:             8,
	ChromaSubsampling: image.YCbCrSubsampleRatio444,
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: genimg <site-dir>")
		os.Exit(2)
	}
	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "genimg:", err)
		os.Exit(1)
	}
}

func run(root string) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(p))
		webp := ext == ".png" || ext == ".gif"
		avifOK := webp || ext == ".jpg" || ext == ".jpeg"
		if !avifOK {
			return nil
		}

		// Decode once and reuse for whichever variants this type supports.
		var img image.Image
		decode := func() (image.Image, error) {
			if img != nil {
				return img, nil
			}
			raw, err := os.ReadFile(p)
			if err != nil {
				return nil, err
			}
			m, _, err := image.Decode(bytes.NewReader(raw))
			if err != nil {
				return nil, err
			}
			img = m
			return img, nil
		}

		if webp {
			if err := variant(p, p+".webp", decode, encodeWebP); err != nil {
				return err
			}
		}
		return variant(p, p+".avif", decode, encodeAVIF)
	})
}

// variant produces dst from src via enc, keeping it only when it beats the
// source. A stale variant that no longer helps is removed so the server never
// embeds a larger one.
func variant(src, dst string, decode func() (image.Image, error), enc func(image.Image) ([]byte, error)) error {
	if upToDate(dst, src) {
		return nil
	}
	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	img, err := decode()
	if err != nil {
		// Not a decodable image (e.g. a malformed file); leave it alone.
		fmt.Fprintf(os.Stderr, "genimg: skip %s: %v\n", src, err)
		return nil
	}
	out, err := enc(img)
	if err != nil {
		fmt.Fprintf(os.Stderr, "genimg: skip %s -> %s: %v\n", src, filepath.Ext(dst), err)
		return nil
	}
	if int64(len(out)) >= si.Size() {
		_ = os.Remove(dst)
		return nil
	}
	if err := os.WriteFile(dst, out, 0o644); err != nil {
		return err
	}
	fmt.Printf("genimg: %s -> %s (%d -> %d, %d%%)\n", src, filepath.Ext(dst), si.Size(), len(out), int64(len(out))*100/si.Size())
	return nil
}

func encodeWebP(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := nativewebp.Encode(&buf, img, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeAVIF(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := avif.Encode(&buf, img, avifOptions); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
