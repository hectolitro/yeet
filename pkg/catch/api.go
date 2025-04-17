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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"strconv"

	"github.com/yeetrun/yeet/pkg/websocketutil"
	"github.com/gorilla/websocket"
	gssh "tailscale.com/tempfork/gliderlabs/ssh"
	"tailscale.com/types/opt"
)

func (s *Server) handleAPI() http.Handler {
	authZ := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := s.verifyCaller(r.Context(), r.RemoteAddr); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			h.ServeHTTP(w, r)
		})
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v0/services", s.handleServices)
	mux.HandleFunc("GET /api/v0/info", s.handleInfo)
	mux.HandleFunc("GET /api/v0/run-command", s.handleRunCommand)
	mux.HandleFunc("GET /api/v0/events", s.handleEvents)
	return authZ(mux)
}

type ServerInfo struct {
	Version string `json:"version"`
	GOOS    string `json:"goos"`
	GOARCH  string `json:"goarch"`
}

func GetInfo() ServerInfo {
	return ServerInfo{
		Version: VersionCommit(),
		GOARCH:  runtime.GOARCH,
		GOOS:    runtime.GOOS,
	}
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(GetInfo())
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleServices(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getServices(w, r)
	case http.MethodPost:
		s.postService(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getServices(w http.ResponseWriter, _ *http.Request) {
	d, err := s.cfg.DB.Get()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(d.AsStruct().Services); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) postService(w http.ResponseWriter, r *http.Request) {

}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func shouldTTY(r *http.Request) bool {
	tty, ttyOK := opt.Bool(r.URL.Query().Get("tty")).Get()
	return !ttyOK || tty
}

type readWriter struct {
	io.Reader
	io.Writer
}

func (s *Server) handleRunCommand(w http.ResponseWriter, r *http.Request) {
	command := r.URL.Query().Get("command")
	tty := shouldTTY(r)
	service := r.URL.Query().Get("service")

	args := r.URL.Query()["args"]
	args = append([]string{command}, args...)

	if command == "" || service == "" {
		http.Error(w, "missing required parameters", http.StatusBadRequest)
		return
	}

	rwc := io.ReadWriter(readWriter{
		Reader: r.Body,
		Writer: w,
	})

	var closer io.Closer
	var ptyReq gssh.Pty
	var ws *websocketutil.ConnReadWriter
	ctx := r.Context()
	if tty {
		rawRows := r.URL.Query().Get("rows")
		rawCols := r.URL.Query().Get("cols")
		if rawRows == "" || rawCols == "" {
			http.Error(w, "missing required parameters", http.StatusBadRequest)
			return
		}
		// Parse rows and cols
		rows, err := strconv.Atoi(rawRows)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cols, err := strconv.Atoi(rawCols)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ptyReq = fakePtyReq(rows, cols)
		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer wsConn.Close()
		closer = wsConn

		ws = websocketutil.NewConnReadWriteCloser(ctx, wsConn)
		defer ws.Close()
		rwc = ws

		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
		go func() {
			<-ws.DoneCh
			cancel()
		}()
	}

	e := &ttyExecer{
		ctx:       ctx,
		s:         s,
		sn:        service,
		user:      "root", // TODO: get user from service
		rawRW:     rwc,
		rawCloser: closer,
		isPty:     tty,
		ptyWCh:    nil, // TODO: implement
		ptyReq:    ptyReq,
		args:      args,
	}

	if ws != nil {
		// Handle PTY resize
		ws.SetReadHandler(func(data []byte) bool {
			if len(data) > 1 && data[0] == 0x01 && string(data[1:4]) == "[8;" {
				resizeMessage := string(data[1:])
				var rows, cols int
				_, err := fmt.Sscanf(resizeMessage, "[8;%d;%dt", &rows, &cols)
				if err != nil {
					log.Println("error parsing resize message:", err)
					return false
				}
				e.ResizeTTY(cols, rows)
				return true
			}
			return false
		})
	}

	if err := e.run(); err != nil {
		log.Println("error running command:", err)
		return
	}
}

func fakePtyReq(rows, cols int) gssh.Pty {
	return gssh.Pty{
		Term: "xterm",
		Window: gssh.Window{
			Width:  cols,
			Height: rows,
		},
	}
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	ch := make(chan Event)
	h := s.AddEventListener(ch, func(et Event) bool {
		return true
	})
	defer s.RemoveEventListener(h)

	for {
		select {
		case event := <-ch:
			conn.WriteJSON(event)
		case <-r.Context().Done():
			return
		}
	}
}
