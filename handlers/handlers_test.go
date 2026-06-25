package handlers

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedirects(t *testing.T) {
	cases := []struct {
		name     string
		handler  http.HandlerFunc
		wantCode int
		wantLoc  string
	}{
		{"RedirectHome", RedirectHome(), http.StatusPermanentRedirect, "/home"},
		{"RedirectConstruction", RedirectConstructionHandler(), http.StatusTemporaryRedirect, "/under-construction"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			c.handler(rr, req)
			if rr.Code != c.wantCode {
				t.Errorf("status = %d, want %d", rr.Code, c.wantCode)
			}
			if got := rr.Header().Get("Location"); got != c.wantLoc {
				t.Errorf("location = %q, want %q", got, c.wantLoc)
			}
		})
	}
}

// TestPageHandlers exercises the fixed-page handlers (which serve a known HTML
// file and may issue server pushes) against a temp static tree.
func TestPageHandlers(t *testing.T) {
	cases := []struct {
		name    string
		page    string
		handler func(int) http.HandlerFunc
		path    string
	}{
		{"Home", "home.html", HomeHandler, "/home"},
		{"Resume", "resume.html", ResumeHomeHandler, "/resume/home"},
		{"HallofArt", "shadow.html", HallofArtHomeHandler, "/hall-of-art/home"},
		{"Chat", "chat.html", ChatHomeHandler, "/chat/home"},
		{"Construction", "construction.html", ConstructionHandler, "/under-construction"},
	}

	for _, c := range cases {
		t.Run(c.name+" serves the page", func(t *testing.T) {
			root := setupStatic(t)
			writeTestFile(t, filepath.Join(root, "html", c.page), "<html>"+c.name+"</html>")

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, c.path, nil)
			c.handler(0)(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rr.Code)
			}
			if got, want := rr.Header().Get("Content-Type"), "text/html; charset=UTF-8"; got != want {
				t.Errorf("content-type = %q, want %q", got, want)
			}
			if !strings.Contains(rr.Body.String(), c.name) {
				t.Errorf("body %q does not contain %q", rr.Body.String(), c.name)
			}
		})

		t.Run(c.name+" missing page is 404", func(t *testing.T) {
			setupStatic(t) // no page file written
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, c.path, nil)
			c.handler(0)(rr, req)
			if rr.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", rr.Code)
			}
		})

		t.Run(c.name+" non-GET is 400", func(t *testing.T) {
			root := setupStatic(t)
			writeTestFile(t, filepath.Join(root, "html", c.page), "x")
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, c.path, nil)
			c.handler(0)(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rr.Code)
			}
		})
	}
}

// TestStaticContentTypes confirms each static handler picks the right Content-Type
// by extension and serves from the expected directory.
func TestStaticContentTypes(t *testing.T) {
	root := setupStatic(t)

	cases := []struct {
		name    string
		sub     string
		file    string
		urlPath string
		wantCT  string
		handler func(int) http.HandlerFunc
	}{
		{"script", "js", "app.js", "/static/js/app.js", "application/javascript; charset=UTF-8", ScriptsHandler},
		{"sourcemap", "js", "app.js.map", "/static/js/app.js.map", "application/json; charset=UTF-8", ScriptsHandler},
		{"css", "css", "home.css", "/static/css/home.css", "text/css; charset=UTF-8", CSSHandler},
		{"html", "html", "frag.html", "/static/html/frag.html", "text/html; charset=UTF-8", HTMLHandler},
		{"typescript", "src", "app.ts", "/static/src/app.ts", "text/plain; charset=UTF-8", TypeScriptHandler},
		{"image", "img", "horse.png", "/static/img/horse.png", "image/png", ImageHandler},
		{"model", "model", "m.gltf", "/static/model/m.gltf", "model/gltf", ModelHandler},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			writeTestFile(t, filepath.Join(root, c.sub, c.file), "data")
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, c.urlPath, nil)
			c.handler(0)(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rr.Code)
			}
			if got := rr.Header().Get("Content-Type"); got != c.wantCT {
				t.Errorf("content-type = %q, want %q", got, c.wantCT)
			}
		})
	}
}

func TestManifestServedFromSiteRoot(t *testing.T) {
	root := setupStatic(t)
	writeTestFile(t, filepath.Join(root, "manifest.json"), `{"name":"shmeeload"}`)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/manifest.json", nil)
	ManifestHandler(0)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if got, want := rr.Header().Get("Content-Type"), "application/json; charset=UTF-8"; got != want {
		t.Errorf("content-type = %q, want %q", got, want)
	}
}
