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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"tailscale.com/client/tailscale"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/util/must"
)

var (
	apiHost = flag.String("api-host", "catch", "API host to connect to")
)

func main() {
	flag.Parse()

	// resolve the fqdn of the api host
	var lc tailscale.LocalClient
	st := must.Get(lc.Status(context.Background()))
	var p *ipnstate.PeerStatus
	for _, sp := range st.Peer {
		if n, _, ok := strings.Cut(sp.DNSName, "."); ok && n == *apiHost {
			p = sp
			break
		}
	}
	if p == nil {
		log.Fatalf("api host %q not found", *apiHost)
	}

	bs, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		log.Fatal(err)
	}
	gitRoot := strings.TrimSpace(string(bs))

	webDir := filepath.Join(string(gitRoot), "pkg/catch/web")
	if st, err := os.Stat(webDir); err != nil || !st.IsDir() {
		log.Fatal(fmt.Errorf("web directory %s does not exist", webDir))
	}

	apiProxy := httputil.NewSingleHostReverseProxy(must.Get(url.Parse("https://" + strings.TrimSuffix(p.DNSName, "."))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		http.FileServer(http.Dir(webDir)).ServeHTTP(w, r)
	})
	http.Handle("/api/v0/", apiProxy)

	fmt.Println("Starting web server at http://localhost:3000")
	must.Do(http.ListenAndServe(":3000", nil))
}
