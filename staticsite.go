package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
)

const (
	mimePNG  = "image/png"
	mimeWebP = "image/webp"
	mimeAVIF = "image/avif"
)

// newMinifier configures a pure-Go minifier for the text asset types the site
// serves. JS is minified per file (the modules import each other by name), so
// exported and imported identifiers are preserved.
func newMinifier() *minify.M {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("image/svg+xml", svg.Minify)
	m.AddFunc("application/json", json.Minify)
	m.AddFunc("text/javascript", js.Minify)
	m.AddFunc("application/javascript", js.Minify)
	return m
}

// minifyBytes returns a minified copy for minifiable text types, or the input
// unchanged when the type is not minifiable, minification errors, or the result
// is not actually smaller.
func minifyBytes(m *minify.M, contentType string, raw []byte) []byte {
	base := contentType
	if i := strings.IndexByte(base, ';'); i >= 0 {
		base = base[:i]
	}
	out, err := m.Bytes(base, raw)
	if err != nil || len(out) == 0 || len(out) >= len(raw) {
		return raw
	}
	return out
}

// variant is an alternate encoding of an asset with its own content-hash ETag.
type variant struct {
	body []byte
	etag string
}

// staticAsset is one served file, prepared once at startup: its content type, a
// content-hash ETag, the raw bytes, precomputed brotli/zstd/gzip variants (for
// compressible types), and WebP/AVIF variants (the build-time "<name>.webp" and
// "<name>.avif" siblings, when generated for an image).
type staticAsset struct {
	contentType string
	etag        string
	raw         []byte
	br          []byte // brotli; nil when not worth it
	zst         []byte // zstd; nil when not worth it
	gz          []byte // gzip; nil when not worth it
	webp        *variant
	avif        *variant
}

// staticSite serves an embedded file tree with ETag revalidation, minified and
// precompressed text (brotli > zstd > gzip), and transparent AVIF/WebP for
// images that shrink.
type staticSite struct {
	assets       map[string]*staticAsset
	cacheControl string
}

func newStaticSite(fsys fs.FS, cacheMaxAge int) (*staticSite, error) {
	s := &staticSite{
		assets:       make(map[string]*staticAsset),
		cacheControl: "public, max-age=" + strconv.Itoa(cacheMaxAge),
	}
	// First pass: read every file. Image variants are precomputed at build time
	// (see cmd/genimg) and embedded as "<name>.webp"/"<name>.avif" siblings, so
	// collect the whole tree before pairing each image with its siblings.
	raw := make(map[string][]byte)
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		raw[p] = b
		return nil
	})
	if err != nil {
		return nil, err
	}

	m := newMinifier()
	for p, b := range raw {
		// Generated image siblings are attached to their source below, not served on their own.
		if strings.HasSuffix(p, ".webp") || strings.HasSuffix(p, ".avif") {
			continue
		}
		ct := mime.TypeByExtension(path.Ext(p))
		if ct == "" {
			ct = http.DetectContentType(b)
		}
		// Minify text assets before hashing and compressing, so the ETag and the
		// br/zstd/gzip variants are all computed from the bytes actually served.
		b = minifyBytes(m, ct, b)
		a := &staticAsset{contentType: ct, etag: etagOf(b), raw: b}
		if compressible(ct) {
			a.br = smaller(brotliBytes(b), b)
			a.zst = smaller(zstdBytes(b), b)
			a.gz = smaller(gzipBytes(b), b)
		}
		// Pair with the build-time image siblings, when present. Their existence
		// already means they beat the source (cmd/genimg only keeps smaller ones).
		if wb, ok := raw[p+".webp"]; ok {
			a.webp = &variant{body: wb, etag: etagOf(wb)}
		}
		if av, ok := raw[p+".avif"]; ok {
			a.avif = &variant{body: av, etag: etagOf(av)}
		}
		s.assets[p] = a
	}
	return s, nil
}

// serve writes the asset at urlPath if present (returning true). For images it
// serves the smallest variant the client accepts (AVIF/WebP over the original);
// otherwise it serves the raw bytes with the best accepted compression. Returns
// false (writing nothing) on a miss.
func (s *staticSite) serve(w http.ResponseWriter, r *http.Request, urlPath string) bool {
	key := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	a, ok := s.assets[key]
	if !ok {
		return false
	}

	h := w.Header()
	h.Set("Cache-Control", s.cacheControl)
	h.Add("Vary", "Accept-Encoding")

	// Image negotiation: serve the smallest variant the client accepts. Each
	// present variant already beats the source, so any accepted one is a win.
	if ct, v := a.bestImage(r); v != nil {
		h.Add("Vary", "Accept")
		h.Set("Content-Type", ct)
		h.Set("ETag", v.etag)
		if r.Header.Get("If-None-Match") == v.etag {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
		writeBody(w, r, v.body)
		return true
	}

	h.Set("Content-Type", a.contentType)
	h.Set("ETag", a.etag)
	if r.Header.Get("If-None-Match") == a.etag {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	body, enc := a.negotiate(r.Header.Get("Accept-Encoding"))
	if enc != "" {
		h.Set("Content-Encoding", enc)
	}
	writeBody(w, r, body)
	return true
}

func writeBody(w http.ResponseWriter, r *http.Request, body []byte) {
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	_, _ = w.Write(body)
}

// negotiate picks the best precomputed compression the client accepts (brotli >
// zstd > gzip), falling back to the raw bytes.
func (a *staticAsset) negotiate(acceptEncoding string) (body []byte, encoding string) {
	accepted := parseList(acceptEncoding)
	switch {
	case a.br != nil && accepted["br"]:
		return a.br, "br"
	case a.zst != nil && accepted["zstd"]:
		return a.zst, "zstd"
	case a.gz != nil && accepted["gzip"]:
		return a.gz, "gzip"
	default:
		return a.raw, ""
	}
}

// bestImage returns the smallest image variant the client accepts, or nil when
// the asset has no variants or the client accepts none of them.
func (a *staticAsset) bestImage(r *http.Request) (contentType string, v *variant) {
	accept := r.Header.Get("Accept")
	if a.avif != nil && strings.Contains(accept, mimeAVIF) {
		contentType, v = mimeAVIF, a.avif
	}
	if a.webp != nil && strings.Contains(accept, mimeWebP) {
		if v == nil || len(a.webp.body) < len(v.body) {
			contentType, v = mimeWebP, a.webp
		}
	}
	return contentType, v
}

func parseList(header string) map[string]bool {
	out := make(map[string]bool)
	for _, tok := range strings.Split(header, ",") {
		name := strings.TrimSpace(strings.SplitN(tok, ";", 2)[0])
		if name != "" {
			out[strings.ToLower(name)] = true
		}
	}
	return out
}

func etagOf(b []byte) string {
	sum := sha256.Sum256(b)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func compressible(contentType string) bool {
	ct := contentType
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	switch {
	case strings.HasPrefix(ct, "text/"):
		return true
	case ct == "application/javascript", ct == "application/json",
		ct == "image/svg+xml", ct == "application/xml", ct == "application/wasm":
		return true
	default:
		return false
	}
}

// smaller returns compressed only when it actually beats raw, else nil.
func smaller(compressed, raw []byte) []byte {
	if len(compressed) > 0 && len(compressed) < len(raw) {
		return compressed
	}
	return nil
}

func gzipBytes(raw []byte) []byte {
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil
	}
	if _, err := zw.Write(raw); err != nil {
		return nil
	}
	if err := zw.Close(); err != nil {
		return nil
	}
	return buf.Bytes()
}

func brotliBytes(raw []byte) []byte {
	var buf bytes.Buffer
	bw := brotli.NewWriterLevel(&buf, brotli.BestCompression)
	if _, err := bw.Write(raw); err != nil {
		return nil
	}
	if err := bw.Close(); err != nil {
		return nil
	}
	return buf.Bytes()
}

func zstdBytes(raw []byte) []byte {
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return nil
	}
	defer enc.Close()
	return enc.EncodeAll(raw, nil)
}
