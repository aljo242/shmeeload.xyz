// Command genimg precomputes smaller image variants under a site directory: a
// lossless WebP for each PNG/GIF, and a lossy AVIF for each PNG/GIF/JPEG. Each
// is written as a "<name>.webp" / "<name>.avif" sibling. The AVIF is kept when it
// beats the source; the (lossless) WebP is kept only when it also beats the AVIF.
// So a photo, where the lossy AVIF wins by a wide margin, ships AVIF + original
// instead of a bloated lossless WebP, while pixel art that WebP packs smaller than
// AVIF keeps its WebP. It runs at build time so the server embeds the variants and
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
	"github.com/gen2brain/avif"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// avifOptionsForExt picks AVIF settings by source type. A JPEG is a photo, where
// q80 at 4:4:4 can't undercut the already-compact source, so it uses 4:2:0 chroma
// at a lower quality: imperceptible on a photo, and it lands well below the JPEG.
// PNG/GIF (icons, pixel art) keep q80 at 4:4:4 so hard edges stay crisp. Speed 8
// keeps the build fast.
func avifOptionsForExt(ext string) avif.Options {
	if ext == ".jpg" || ext == ".jpeg" {
		return avif.Options{
			Quality:           60,
			QualityAlpha:      60,
			Speed:             8,
			ChromaSubsampling: image.YCbCrSubsampleRatio420,
		}
	}
	return avif.Options{
		Quality:           80,
		QualityAlpha:      80,
		Speed:             8,
		ChromaSubsampling: image.YCbCrSubsampleRatio444,
	}
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

		si, err := os.Stat(p)
		if err != nil {
			return err
		}

		// AVIF first: its kept size is the bar the lossless WebP must also clear,
		// so a photo's huge lossless WebP is dropped in favor of the tiny AVIF.
		avifOpts := avifOptionsForExt(ext)
		avifSize, err := variant(p, p+".avif", decode, func(img image.Image) ([]byte, error) {
			return encodeAVIF(img, avifOpts)
		}, si.Size())
		if err != nil {
			return err
		}
		if webp {
			beat := si.Size()
			if avifSize > 0 && avifSize < beat {
				beat = avifSize
			}
			if _, err := variant(p, p+".webp", decode, encodeWebP, beat); err != nil {
				return err
			}
		}
		return nil
	})
}

// variant produces dst from src via enc, keeping it only when its size is under
// beat (the source size, or for WebP the smaller of source and AVIF). It returns
// the kept size, or -1 when nothing was written. A stale variant that no longer
// helps is removed so the server never embeds a larger one.
func variant(src, dst string, decode func() (image.Image, error), enc func(image.Image) ([]byte, error), beat int64) (int64, error) {
	if upToDate(dst, src) {
		if di, err := os.Stat(dst); err == nil {
			return di.Size(), nil
		}
		return -1, nil
	}
	img, err := decode()
	if err != nil {
		// Not a decodable image (e.g. a malformed file); leave it alone.
		fmt.Fprintf(os.Stderr, "genimg: skip %s: %v\n", src, err)
		return -1, nil
	}
	out, err := enc(img)
	if err != nil {
		fmt.Fprintf(os.Stderr, "genimg: skip %s -> %s: %v\n", src, filepath.Ext(dst), err)
		return -1, nil
	}
	if int64(len(out)) >= beat {
		_ = os.Remove(dst)
		return -1, nil
	}
	if err := os.WriteFile(dst, out, 0o644); err != nil {
		return -1, err
	}
	fmt.Printf("genimg: %s -> %s (%d bytes, under %d)\n", src, filepath.Ext(dst), len(out), beat)
	return int64(len(out)), nil
}

func encodeWebP(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := nativewebp.Encode(&buf, img, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeAVIF(img image.Image, opts avif.Options) ([]byte, error) {
	var buf bytes.Buffer
	if err := avif.Encode(&buf, img, opts); err != nil {
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
