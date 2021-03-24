package handlers

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/aljo242/http_util"
	"github.com/rs/zerolog/log"
)

const (
	htmlDir string = "./static/html/"
	jsDir   string = "./static/js/"
	cssDir  string = "./static/css/"
	tsDir   string = "./static/src/"
	imgDir  string = "./static/img/"
)

// ScriptsHandler takes a script name and
func ScriptsHandler(scriptName string, debugEnable bool) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Base(r.URL.Path)
		log.Debug().Str("Handler", "ScriptsHandler").Str("Filename", filename).Msg("incoming request")
		if r.Method == http.MethodGet {
			wantFile := filepath.Join(jsDir, filename)
			if _, err := os.Stat(wantFile); os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
				log.Fatal().Err(err).Str("Filename", wantFile).Msg("Error finding file")
				return
			}

			//w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
			http.ServeFile(w, r, wantFile)
			//if debugEnable {
			//	http.ServeFile(w, r, filepath.Join(jsDir, "../src/app.ts"))
			//}

		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// CSSHandler takes a script name and
func CSSHandler(filename string, debugEnable bool) func(http.ResponseWriter, *http.Request) {
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
			//w.WriteHeader(http.StatusOK)
			http.ServeFile(w, r, wantFile)

		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// HTMLHandler takes a script name and
func HTMLHandler(scriptName string, debugEnable bool) func(http.ResponseWriter, *http.Request) {
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
			http.ServeFile(w, r, wantFile)

		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// TypeScriptHandler takes a script name and returns a HandleFunc
func TypeScriptHandler(scriptName string, debugEnable bool) func(http.ResponseWriter, *http.Request) {
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
			http.ServeFile(w, r, wantFile)

		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// ImageHandler returns a HandleFunc to serve image files
func ImageHandler() func(http.ResponseWriter, *http.Request) {
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

			switch filepath.Ext(filename) {
			case ".jpg", ".jpeg":
				w.Header().Set("Content-Type", "image/jpeg")
			case ".png":
				w.Header().Set("Content-Type", "image/png")
			case ".gif":
				w.Header().Set("Content-Type", "image/gif")
			case ".ico":
				w.Header().Set("Content-Type", "image/x-icon")
			}
			http.ServeFile(w, r, wantFile)

		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// HomeHandler serves the home.html file
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	//http_util.CheckHTTP2Support(w)
	// this page currently only serves html resources
	log.Debug().Str("Handler", "HomeHandler").Msg("incoming request")

	if r.Method == http.MethodGet {
		defer func() {
			wantFile := filepath.Join(htmlDir, "home.html")
			if _, err := os.Stat(wantFile); os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
				log.Debug().Err(err).Str("Filename", wantFile).Msg("Error finding file")
				return
			}

			//w.WriteHeader(http.StatusOK)
			http.ServeFile(w, r, wantFile)
		}()

		pusher, ok := w.(http.Pusher)
		if ok {
			// push js file
			//wantFile := filepath.Join(jsDir, "app.js")
			wantFile := jsDir + "app.js"
			if _, err := os.Stat(wantFile); os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
				log.Fatal().Err(err).Str("Filename", wantFile).Msg("Error finding file")
				return
			}
			absWantFile, _ := filepath.Abs(wantFile)
			log.Debug().Str("file", absWantFile).Msg("pushing file")
			err := pusher.Push(absWantFile, nil)
			if err != nil {
				log.Debug().Err(err).Str("Filename", wantFile).Msg("Error pushing file")
				return
			}

			// push css file
			//wantFile = filepath.Join(cssDir, "home.css")
			wantFile = cssDir + "home.css"
			if _, err := os.Stat(wantFile); os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
				log.Fatal().Err(err).Str("Filename", wantFile).Msg("Error finding file")
				return
			}
			absWantFile, _ = filepath.Abs(wantFile)
			log.Debug().Str("file", absWantFile).Msg("pushing file")
			err = pusher.Push(absWantFile, nil)
			if err != nil {
				log.Debug().Err(err).Str("Filename", wantFile).Msg("Error pushing file")
				return
			}

			// push favicon
			//wantFile = filepath.Join(imgDir, "favicon.ico")
			wantFile = imgDir + "favicon.ico"
			if _, err := os.Stat(wantFile); os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
				log.Fatal().Err(err).Str("Filename", wantFile).Msg("Error finding file")
				return
			}
			absWantFile, _ = filepath.Abs(wantFile)
			log.Debug().Str("file", absWantFile).Msg("pushing file")
			err = pusher.Push(absWantFile, nil)
			if err != nil {
				log.Debug().Err(err).Str("Filename", wantFile).Msg("Error pushing file")
				return
			}
		}

	} else {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

// RedirectHome redirects urls to the address to be served by HomeHandler
func RedirectHome(host string, debugEnable bool) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		home := host + "/home"
		log.Debug().Str("Handler", "RedirectHome").Str("Home", home).Msg("incoming request")
		http.Redirect(w, r, "/home", http.StatusFound)
	}
}

// ChatHomeHandler is the route for the chat home where users can get assigned unique identifiers
func ChatHomeHandler(filename string, debugEnable bool) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		// this page currently only serves html resources
		log.Debug().Str("Handler", "ChatHomeHandler").Msg("incoming request")

		if r.Method == http.MethodGet {
			//wantFile := filepath.Join(htmlDir, "chat.html")
			//if _, err := os.Stat(wantFile); os.IsNotExist(err) {
			//	w.WriteHeader(http.StatusNotFound)
			//	log.Fatalf("Error finding file %v : %v", wantFile, err)
			//	return
			//}

			//w.WriteHeader(http.StatusOK)
			//http.ServeFile(w, r, wantFile)

			defer func() {
				wantFile := filepath.Join(htmlDir, "chat.html")
				if _, err := os.Stat(wantFile); os.IsNotExist(err) {
					w.WriteHeader(http.StatusNotFound)
					log.Fatal().Err(err).Str("Filename", wantFile).Msg("Error finding file")
					return
				}

				//w.WriteHeader(http.StatusOK)
				http.ServeFile(w, r, wantFile)
			}()

			pusher, ok := w.(http.Pusher)
			if ok {
				// push js file
				//wantFile := filepath.Join(jsDir, "app.js")
				wantFile := jsDir + "chat.js"
				if _, err := os.Stat(wantFile); os.IsNotExist(err) {
					w.WriteHeader(http.StatusNotFound)
					log.Fatal().Err(err).Str("Filename", wantFile).Msg("Error finding file")
					return
				}
				absWantFile, _ := filepath.Abs(wantFile)
				log.Debug().Str("file", absWantFile).Msg("pushing file")
				err := pusher.Push(absWantFile, nil)
				if err != nil {
					log.Fatal().Err(err).Str("Filename", wantFile).Msg("Error pushing file")
					return
				}

				// push css file
				//wantFile = filepath.Join(cssDir, "home.css")
				wantFile = cssDir + "chat.css"
				if _, err := os.Stat(wantFile); os.IsNotExist(err) {
					w.WriteHeader(http.StatusNotFound)
					log.Fatal().Err(err).Str("Filename", wantFile).Msg("Error finding file")
					return
				}
				absWantFile, _ = filepath.Abs(wantFile)
				log.Debug().Str("file", absWantFile).Msg("pushing file")
				err = pusher.Push(absWantFile, nil)
				if err != nil {
					log.Fatal().Err(err).Str("Filename", wantFile).Msg("Error pushing file")
					return
				}
			}

			wantFile := cssDir + "chat.css"
			chatfilepath, _ := filepath.Abs(wantFile)
			err := http_util.PushFiles(w, chatfilepath)
			if err != nil {
				log.Fatal().Err(err).Msg("Error pushing files")
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

// ResumeHomeHandler takes a script name and
func ResumeHomeHandler(debugEnable bool) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		filename := filepath.Base(r.URL.Path)
		log.Debug().Str("Handler", "ResumeHomeHandler").Str("Filename", filename).Msg("incoming request")

		if r.Method == http.MethodGet {
			wantFile := filepath.Join(htmlDir, "resumeHome.html")
			if _, err := os.Stat(wantFile); os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
				log.Fatal().Err(err).Str("Filename", wantFile).Msg("Error finding file")
				return
			}

			//w.WriteHeader(http.StatusOK)
			http.ServeFile(w, r, wantFile)

		} else {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}
