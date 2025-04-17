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
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"github.com/yeetrun/yeet/pkg/db"
	"github.com/yeetrun/yeet/pkg/svc"
	"github.com/yeetrun/yeet/tempfork/google/go-containerregistry/pkg/registry"
	"tailscale.com/util/mak"
)

func (s *Server) newRegistry() *containerRegistry {
	bd := filepath.Join(s.cfg.RegistryRoot, "blobs")
	md := filepath.Join(s.cfg.RegistryRoot, "manifests")
	if err := os.MkdirAll(bd, 0700); err != nil {
		log.Fatalf("MkdirAll: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(md, "sha256"), 0700); err != nil {
		log.Fatalf("MkdirAll: %v", err)
	}
	bh := registry.NewDiskBlobHandler(filepath.Join(s.cfg.RegistryRoot, "blobs"))
	cr := &containerRegistry{
		s:           s,
		manifestDir: md,
	}
	cr.r = registry.New(
		registry.WithBlobHandler(bh),
		registry.WithCallbackHandler(cr),
		registry.WithManifestHandler(cr),
	)
	return cr
}

type containerRegistry struct {
	s *Server

	manifestDir string
	r           http.Handler
}

func (cr *containerRegistry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only allow read-only access to the registry from localhost.
	if ap, err := netip.ParseAddrPort(r.RemoteAddr); err != nil {
		log.Printf("ParseAddrPort: %v", err)
		http.Error(w, "Registry is read-only", http.StatusMethodNotAllowed)
		return
	} else if ap.Addr().IsLoopback() {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			http.Error(w, "Registry is read-only", http.StatusMethodNotAllowed)
			return
		}
	} else {
		if err := cr.s.verifyCaller(r.Context(), r.RemoteAddr); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
	}
	cr.r.ServeHTTP(w, r)
}

func (cr *containerRegistry) AllRepos() []string {
	log.Printf("AllManifests")
	dv, err := cr.s.getDB()
	if err != nil {
		log.Printf("getDB: %v", err)
		return nil
	}
	var out []string
	for rn := range dv.Images().All() {
		out = append(out, string(rn))
	}
	return out
}

func (cr *containerRegistry) RepoExists(repo string) bool {
	dv, err := cr.s.getDB()
	if err != nil {
		log.Printf("getDB: %v", err)
		return false
	}
	_, ok := dv.Images().GetOk(db.ImageRepoName(repo))
	return ok
}

func (cr *containerRegistry) Manifests(repo string) (map[string]registry.Manifest, bool) {
	log.Printf("Manifests: %s", repo)
	dv, err := cr.s.getDB()
	if err != nil {
		log.Printf("getDB: %v", err)
		return nil, false
	}
	ir, ok := dv.Images().GetOk(db.ImageRepoName(repo))
	if !ok {
		return nil, false
	}
	if ir.Refs().Len() == 0 {
		return nil, true
	}
	x := map[string]registry.Manifest{}
	for tag, m := range ir.Refs().All() {
		mb, err := cr.readManifest(m.BlobHash)
		if err != nil {
			log.Printf("readManifest: %v", err)
			continue
		}
		x[string(tag)] = registry.Manifest{
			ContentType: m.ContentType,
			Blob:        mb,
		}
	}
	return x, true
}

func (cr *containerRegistry) Manifest(repo, reference string) (registry.Manifest, bool) {
	log.Printf("Manifest: %s %s", repo, reference)
	dv, err := cr.s.getDB()
	if err != nil {
		log.Printf("getDB: %v", err)
		return registry.Manifest{}, false
	}
	ir, ok := dv.Images().GetOk(db.ImageRepoName(repo))
	if !ok {
		return registry.Manifest{}, false
	}
	m, ok := ir.Refs().GetOk(db.ImageRef(reference))
	if !ok {
		return registry.Manifest{}, false
	}
	mb, err := cr.readManifest(m.BlobHash)
	if err != nil {
		log.Printf("readManifest: %v", err)
		return registry.Manifest{}, false
	}
	return registry.Manifest{
		ContentType: m.ContentType,
		Blob:        mb,
	}, true
}

func (cr *containerRegistry) storeManifest(b []byte) (string, error) {
	sha := fmt.Sprintf("%x", sha256.Sum256(b))
	if err := os.WriteFile(filepath.Join(cr.manifestDir, "sha256", sha), b, 0600); err != nil {
		return "", fmt.Errorf("storeManifest: %w", err)
	}
	return sha, nil
}

func (cr *containerRegistry) readManifest(sha256 string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(cr.manifestDir, "sha256", sha256))
	if err != nil {
		return nil, fmt.Errorf("readManifest(%q): %w", sha256, err)
	}
	return b, nil
}

func (cr *containerRegistry) DeleteManifest(repo, ref string) {
	log.Printf("DeleteManifest: %s %s", repo, ref)
	dv, err := cr.s.getDB()
	if err != nil {
		log.Printf("getDB: %v", err)
		return
	}
	d := dv.AsStruct()
	ir, ok := d.Images[db.ImageRepoName(repo)]
	if !ok {
		return
	}
	delete(ir.Refs, db.ImageRef(ref))
	if err := cr.s.cfg.DB.Set(d); err != nil {
		log.Printf("Set: %v", err)
	}
}

func (cr *containerRegistry) SetManifest(repo, tag string, manifest registry.Manifest) {
	log.Printf("SetManifest: %s %s %v", repo, tag, manifest)
	if strings.Count(repo, "/") != 1 {
		// If the repo is not in the format of 'service/container', it's invalid.
		return
	}

	svcName := repo
	if svc, container, ok := strings.Cut(repo, "/"); !ok {
		// If not ok
		log.Printf("containers should follow the 'service/container' format")
		return
	} else {
		// If ok
		if strings.Contains(container, "/") {
			log.Printf("invalid container name: %q", container)
			return
		}
		svcName = svc
	}
	var references []string
	var shouldInstall bool
	switch tag {
	case "run":
		// "run" == auto-deploy image, so we should install it.
		references = []string{"run", "staged"}
		shouldInstall = true
	case "latest":
		// We accept "latest" as a tag, but we store it as "staged".
		references = []string{"staged"}
	default:
		references = []string{tag}
	}
	dv, err := cr.s.getDB()
	if err != nil {
		log.Printf("getDB: %v", err)
		return
	}
	d := dv.AsStruct()
	ir, ok := d.Images[db.ImageRepoName(repo)]
	if !ok {
		ir = &db.ImageRepo{
			Refs: make(map[db.ImageRef]db.ImageManifest, 1),
		}
		mak.Set(&d.Images, db.ImageRepoName(repo), ir)
	}
	mh, err := cr.storeManifest(manifest.Blob)
	if err != nil {
		log.Printf("storeManifest: %v", err)
		return
	}
	for _, reference := range references {
		ir.Refs[db.ImageRef(reference)] = db.ImageManifest{
			ContentType: manifest.ContentType,
			BlobHash:    mh,
		}
	}
	if err := cr.s.cfg.DB.Set(d); err != nil {
		log.Printf("Set: %v", err)
		return
	}
	image := fmt.Sprintf("%s/%s", svc.InternalRegistryHost, repo)

	// TODO: remove FileInstaller, use the new Installer directly.
	inst, err := NewFileInstaller(cr.s, FileInstallerCfg{
		InstallerCfg: InstallerCfg{
			ServiceName: svcName,
			ClientOut:   io.Discard,
			Printer:     log.Printf,
		},
		StageOnly: !shouldInstall,
	})
	if err != nil {
		log.Printf("NewFileInstaller: %v", err)
		return
	}
	defer inst.Close()

	// Check if previous generation compose file exists and copy it if found
	var composeFile string
	if svc, ok := d.Services[svcName]; ok && svc.Generation > 0 {
		prevGen := svc.Generation - 1
		if prevFile, ok := svc.Artifacts.Gen(db.ArtifactDockerComposeFile, prevGen); ok {
			// Previous compose file exists, copy it to the new generation
			content, err := os.ReadFile(prevFile)
			if err != nil {
				log.Printf("failed to read previous generation compose file: %v", err)
				inst.Fail()
				return
			}
			composeFile = string(content)
		}
	}

	// If no previous file found or couldn't read it, use template
	if composeFile == "" {
		composeFile = fmt.Sprintf(composeTemplate, svcName, image, cr.s.serviceDataDir(svcName))
	}

	if _, err := io.Copy(inst, strings.NewReader(composeFile)); err != nil {
		inst.Fail()
		log.Printf("failed to write compose file: %v", err)
		return
	}
	if err := inst.Close(); err != nil {
		log.Printf("failed to close installer: %v", err)
	}
}

func (cr *containerRegistry) SetManifests(repo string, manifests map[string]registry.Manifest) {
	log.Printf("SetManifests: %s %v", repo, manifests)
	dv, err := cr.s.getDB()
	if err != nil {
		log.Printf("getDB: %v", err)
		return
	}
	d := dv.AsStruct()
	ir, ok := d.Images[db.ImageRepoName(repo)]
	if !ok {
		ir = &db.ImageRepo{
			Refs: make(map[db.ImageRef]db.ImageManifest, len(manifests)),
		}
		mak.Set(&d.Images, db.ImageRepoName(repo), ir)
	}
	for reference, manifest := range manifests {
		mh, err := cr.storeManifest(manifest.Blob)
		if err != nil {
			log.Printf("storeManifest: %v", err)
			return
		}
		ir.Refs[db.ImageRef(reference)] = db.ImageManifest{
			ContentType: manifest.ContentType,
			BlobHash:    mh,
		}
	}
	if err := cr.s.cfg.DB.Set(d); err != nil {
		log.Printf("Set: %v", err)
	}
}

const composeTemplate = `services:
  %s:
    image: %s
    restart: unless-stopped
    volumes:
      - %s:/data
`

func (cr *containerRegistry) OnImageReceived(repo, tag, digest string) error {
	log.Printf("OnImageReceived: %s %s %s", repo, tag, digest)

	if strings.Count(repo, "/") != 1 {
		// If the repo is not in the format of 'service/container', it's invalid.
		return errors.New("containers should follow the 'service/container' format")
	}

	if tag != "latest" && tag != "run" {
		return fmt.Errorf("invalid tag: %q", tag)
	}

	return nil
}
