// Copyright 2025 AUTHORS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package catch

import (
	"embed"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed web/*
var webFS embed.FS

func (s *Server) WebMux() (http.Handler, error) {
	mux := http.NewServeMux()

	webRoot, err := fs.Sub(webFS, "web")
	if err != nil {
		return nil, err
	}
	fileHandler := http.FileServer(http.FS(webRoot))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filePath := strings.TrimPrefix(r.URL.Path, "/")
		if filePath == "" {
			filePath = "index.html"
		}
		file, err := webRoot.Open(filePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Generate ETag based on file modification time and size
		etag := `W/"` + fileInfo.ModTime().Format(time.RFC3339) + `-` + strconv.FormatInt(fileInfo.Size(), 10) + `"`
		w.Header().Set("ETag", etag)

		if match := r.Header.Get("If-None-Match"); match != "" {
			if match == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		fileHandler.ServeHTTP(w, r)
	}))

	// The registry handler is mounted at /v2/.
	mux.Handle("/v2/", s.registry)
	// Mount the API handler at /api/v0/.
	mux.Handle("/api/v0/", s.handleAPI())
	return mux, nil
}
