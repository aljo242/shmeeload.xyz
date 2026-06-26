package main

import (
	"embed"
	"io/fs"
)

// siteEmbed holds the entire served site, baked into the binary at build time.
// site/static/js is produced by the TypeScript build before `go build` runs.
//
//go:embed site
var siteEmbed embed.FS

// siteFS returns the embedded content rooted at the site/ directory.
func siteFS() fs.FS {
	sub, err := fs.Sub(siteEmbed, "site")
	if err != nil {
		// The embed path is a compile-time constant, so this cannot fail.
		panic(err)
	}
	return sub
}
