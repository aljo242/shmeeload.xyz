package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/aljo242/chef"
	"github.com/rs/zerolog/log"
)

const (
	htmlDir string = "./static/html/"
	jsDir   string = "./static/js/"
	cssDir  string = "./static/css/"
	tsDir   string = "./static/src/"
	imgDir  string = "./static/img/"
	rootDir string = "./"
)

// ScriptsHandler takes a script name and
func ScriptsHandler(cacheMaxAge int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Base(r.URL.Path)
		log.Debug().Str("Handler", "ScriptsHandler").Str("Filename", filename).Msg("incoming request")
		if r.Method == http.MethodGet {
			wantFile := filepath.Join(jsDir, filename)
			if _, err := os.Stat(wantFile); os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
				log.Debug().Err(err).Str("Filename", wantFile).Msg("Error finding file")
				return
			}

			switch filepath.Ext(wantFile) {
			case ".js":
				w.Header().Set("Content-Type", "application/javascript; charset=UTF-8")
			case ".js.map":
				w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			}

			w.Header().Set("Cache-Control", "max-age="+strconv.FormatInt(int64(cacheMaxAge), 10))
			http.ServeFile(w, r, wantFile)

		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// CSSHandler takes a script name and
func CSSHandler(cacheMaxAge int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Base(r.URL.Path)
		log.Debug().Str("Handler", "CSSHandler").Str("Filename", filename).Msg("incoming request")

		if r.Method == http.MethodGet {
			wantFile := filepath.Join(cssDir, filename)
			if _, err := os.Stat(wantFile); os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
				log.Debug().Err(err).Str("Filename", wantFile).Msg("Error finding file")
				return
			}

			w.Header().Set("Content-Type", "text/css; charset=UTF-8")
			w.Header().Set("Cache-Control", "max-age="+strconv.FormatInt(int64(cacheMaxAge), 10))
			http.ServeFile(w, r, wantFile)

		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// HTMLHandler takes a script name and
func HTMLHandler(cacheMaxAge int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Base(r.URL.Path)
		log.Debug().Str("Handler", "HTMLHandler").Str("Filename", filename).Msg("incoming request")

		if r.Method == http.MethodGet {
			wantFile := filepath.Join(htmlDir, filename)
			if _, err := os.Stat(wantFile); os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
				log.Debug().Err(err).Str("Filename", wantFile).Msg("Error finding file")
				return
			}

			//w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			w.Header().Set("Cache-Control", "max-age="+strconv.FormatInt(int64(cacheMaxAge), 10))
			http.ServeFile(w, r, wantFile)

		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// TypeScriptHandler takes a script name and returns a HandleFunc
func TypeScriptHandler(cacheMaxAge int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Base(r.URL.Path)
		log.Debug().Str("Handler", "TypeScriptHandler").Str("Filename", filename).Msg("incoming request")

		if r.Method == http.MethodGet {
			wantFile := filepath.Join(tsDir, filename)
			if _, err := os.Stat(wantFile); os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
				log.Debug().Err(err).Str("Filename", wantFile).Msg("Error finding file")
				return
			}

			//w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
			w.Header().Set("Cache-Control", "max-age="+strconv.FormatInt(int64(cacheMaxAge), 10))
			http.ServeFile(w, r, wantFile)

		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// ManifestHandler serves manifest.json
func ManifestHandler(cacheMaxAge int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Base(r.URL.Path)

		if r.Method == http.MethodGet {
			log.Debug().Str("Handler", "ManifestHandler").Str("Filename", filename).Msg("incoming request")
			wantFile := filepath.Join(rootDir, filename)
			if _, err := os.Stat(wantFile); os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
				log.Debug().Err(err).Str("Filename", wantFile).Msg("Error finding file")
				return
			}

			//w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.Header().Set("Cache-Control", "max-age="+strconv.FormatInt(int64(cacheMaxAge), 10))
			http.ServeFile(w, r, wantFile)

		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// ServiceWorkerHandler serves serviceWorker.js
func ServiceWorkerHandler(cacheMaxAge int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Base(r.URL.Path)

		if r.Method == http.MethodGet {
			log.Debug().Str("Handler", "ServiceWorkerHandler").Str("Filename", filename).Msg("incoming request")
			wantFile := filepath.Join(rootDir, filename)
			if _, err := os.Stat(wantFile); os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
				log.Debug().Err(err).Str("Filename", wantFile).Msg("Error finding file")
				return
			}
			switch filepath.Ext(wantFile) {
			case ".js":
				w.Header().Set("Content-Type", "application/javascript; charset=UTF-8")
			case ".js.map":
				w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			}
			w.Header().Set("Cache-Control", "max-age="+strconv.FormatInt(int64(cacheMaxAge), 10))
			http.ServeFile(w, r, wantFile)

		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// ImageHandler returns a HandleFunc to serve image files
func ImageHandler(cacheMaxAge int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Base(r.URL.Path)

		if r.Method == http.MethodGet {
			wantFile := filepath.Join(imgDir, filename)
			log.Debug().Str("Handler", "ImageHandler").Str("Filename", wantFile).Msg("incoming request")

			if _, err := os.Stat(wantFile); os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
				log.Debug().Err(err).Str("Filename", wantFile).Msg("Error finding file")
				return
			}

			switch filepath.Ext(wantFile) {
			case ".jpg", ".jpeg":
				w.Header().Set("Content-Type", "image/jpeg")
			case ".png":
				w.Header().Set("Content-Type", "image/png")
			case ".gif":
				w.Header().Set("Content-Type", "image/gif")
			case ".ico":
				w.Header().Set("Content-Type", "image/x-icon")
			}
			w.Header().Set("Cache-Control", "max-age="+strconv.FormatInt(int64(cacheMaxAge), 10))
			http.ServeFile(w, r, wantFile)

		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// RedirectHome redirects to the {HOST}/home url with a 301 status
func RedirectHome() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug().Str("Handler", "RedirectHome").Str("Request URL", r.URL.Path).Msg("incoming request")
		http.Redirect(w, r, r.URL.Host+"/home", http.StatusPermanentRedirect)
	}
}

// RedirectConstructionHandler redirects to the {HOST}/under-construction url (construction handler) with a 307 (temporary moved) status
func RedirectConstructionHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug().Str("Handler", "RedirectConstructionHandler").Str("Request URL", r.URL.Path).Msg("incoming request")
		http.Redirect(w, r, r.URL.Host+"/under-construction", http.StatusTemporaryRedirect)
	}
}

func ConstructionHandler(cacheMaxAge int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			log.Debug().Str("Handler", "ConstructionHandler").Msg("incoming request")
			defer func() {
				wantFile := filepath.Join(htmlDir, "construction.html")
				if _, err := os.Stat(wantFile); os.IsNotExist(err) {
					w.WriteHeader(http.StatusNotFound)
					log.Debug().Err(err).Str("Filename", wantFile).Msg("Error finding file")
					return
				}

				//w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "text/html; charset=UTF-8")
				w.Header().Set("Cache-Control", "max-age="+strconv.FormatInt(int64(cacheMaxAge), 10))
				http.ServeFile(w, r, wantFile)
			}()
		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// HomeHandler serves the home.html file
func HomeHandler(cacheMaxAge int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method == http.MethodGet {
			log.Debug().Str("Handler", "HomeHandler").Msg("incoming request")
			defer func() {
				wantFile := filepath.Join(htmlDir, "home.html")
				if _, err := os.Stat(wantFile); os.IsNotExist(err) {
					w.WriteHeader(http.StatusNotFound)
					log.Debug().Err(err).Str("Filename", wantFile).Msg("Error finding file")
					return
				}

				//w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "text/html; charset=UTF-8")
				w.Header().Set("Cache-Control", "max-age="+strconv.FormatInt(int64(cacheMaxAge), 10))
				http.ServeFile(w, r, wantFile)
			}()

			wantFile := cssDir + "home.css"
			chatFilepath, _ := filepath.Abs(wantFile)
			wantFile = jsDir + "app.js"
			jsFilepath, _ := filepath.Abs(wantFile)
			wantFile = imgDir + "1favicon.ico"
			faviconFilepath, _ := filepath.Abs(wantFile)
			err := chef.PushFiles(w, chatFilepath, jsFilepath, faviconFilepath)
			if err != nil {
				log.Error().Err(err).Msg("Error pushing files")
			}

		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// ResumeHomeHandler takes a script name and
func ResumeHomeHandler(cacheMaxAge int) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Base(r.URL.Path)
		log.Debug().Str("Handler", "ResumeHomeHandler").Str("Filename", filename).Msg("incoming request")

		if r.Method == http.MethodGet {
			defer func() {
				wantFile := filepath.Join(htmlDir, "resume.html")
				if _, err := os.Stat(wantFile); os.IsNotExist(err) {
					w.WriteHeader(http.StatusNotFound)
					log.Fatal().Err(err).Str("Filename", wantFile).Msg("Error finding file")
					return
				}

				w.Header().Set("Content-Type", "text/html; charset=UTF-8")
				w.Header().Set("Cache-Control", "max-age="+strconv.FormatInt(int64(cacheMaxAge), 10))
				http.ServeFile(w, r, wantFile)
			}()

			wantFile := cssDir + "resume.css"
			chatFilepath, _ := filepath.Abs(wantFile)

			err := chef.PushFiles(w, chatFilepath)
			if err != nil {
				log.Error().Err(err).Msg("Error pushing files")
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// ResumeHomeHandler takes a script name and
func TunesHomeHandler(cacheMaxAge int) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Base(r.URL.Path)
		log.Debug().Str("Handler", "ResumeHomeHandler").Str("Filename", filename).Msg("incoming request")

		if r.Method == http.MethodGet {
			defer func() {
				wantFile := filepath.Join(htmlDir, "resume.html")
				if _, err := os.Stat(wantFile); os.IsNotExist(err) {
					w.WriteHeader(http.StatusNotFound)
					log.Fatal().Err(err).Str("Filename", wantFile).Msg("Error finding file")
					return
				}

				w.Header().Set("Content-Type", "text/html; charset=UTF-8")
				w.Header().Set("Cache-Control", "max-age="+strconv.FormatInt(int64(cacheMaxAge), 10))
				http.ServeFile(w, r, wantFile)
			}()

			wantFile := cssDir + "resume.css"
			chatFilepath, _ := filepath.Abs(wantFile)

			err := chef.PushFiles(w, chatFilepath)
			if err != nil {
				log.Error().Err(err).Msg("Error pushing files")
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}
