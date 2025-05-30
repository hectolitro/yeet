// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	v1 "github.com/yeetrun/yeet/tempfork/google/go-containerregistry/pkg/v1"
	"github.com/yeetrun/yeet/tempfork/google/go-containerregistry/pkg/v1/types"
)

type catalog struct {
	Repos []string `json:"repositories"`
}

type listTags struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type ManifestHandler interface {
	AllRepos() []string
	RepoExists(repo string) bool
	Manifests(repo string) (map[string]Manifest, bool)
	Manifest(repo, target string) (Manifest, bool)
	SetManifests(repo string, manifests map[string]Manifest)
	SetManifest(repo, target string, manifest Manifest)
	DeleteManifest(repo, target string)
}

type memManifestHandler struct {
	lock sync.RWMutex
	m    map[string]map[string]Manifest
}

func (m *memManifestHandler) AllRepos() []string {
	m.lock.RLock()
	defer m.lock.RUnlock()
	var repos []string
	for repo := range m.m {
		repos = append(repos, repo)
	}
	return repos
}

func (m *memManifestHandler) Manifests(repo string) (map[string]Manifest, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	x, ok := m.m[repo]
	return x, ok
}

func (m *memManifestHandler) RepoExists(repo string) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, ok := m.m[repo]
	return ok
}

func (m *memManifestHandler) Manifest(repo, target string) (Manifest, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	c, ok := m.m[repo]
	if !ok {
		return Manifest{}, false
	}
	mf, ok := c[target]
	return mf, ok
}

func (m *memManifestHandler) SetManifests(repo string, manifests map[string]Manifest) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.m[repo] = manifests
}

func (m *memManifestHandler) SetManifest(repo, target string, mfest Manifest) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, ok := m.m[repo]; !ok {
		m.m[repo] = make(map[string]Manifest)
	}
	m.m[repo][target] = mfest
}

func (m *memManifestHandler) DeleteManifest(repo, target string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.m[repo], target)
}

type Manifest struct {
	ContentType string
	Blob        []byte
}

type manifests struct {
	// maps repo -> manifest tag/digest -> manifest
	manifestHandler ManifestHandler
	log             *log.Logger
}

func isManifest(req *http.Request) bool {
	elems := strings.Split(req.URL.Path, "/")
	elems = elems[1:]
	if len(elems) < 4 {
		return false
	}
	return elems[len(elems)-2] == "manifests"
}

func isTags(req *http.Request) bool {
	elems := strings.Split(req.URL.Path, "/")
	elems = elems[1:]
	if len(elems) < 4 {
		return false
	}
	return elems[len(elems)-2] == "tags"
}

func isCatalog(req *http.Request) bool {
	elems := strings.Split(req.URL.Path, "/")
	elems = elems[1:]
	if len(elems) < 2 {
		return false
	}

	return elems[len(elems)-1] == "_catalog"
}

// Returns whether this url should be handled by the referrers handler
func isReferrers(req *http.Request) bool {
	elems := strings.Split(req.URL.Path, "/")
	elems = elems[1:]
	if len(elems) < 4 {
		return false
	}
	return elems[len(elems)-2] == "referrers"
}

// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#pulling-an-image-manifest
// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#pushing-an-image
func (m *manifests) handle(resp http.ResponseWriter, req *http.Request, cbh CallbackHandler) *regError {
	elem := strings.Split(req.URL.Path, "/")
	elem = elem[1:]
	target := elem[len(elem)-1]
	repo := strings.Join(elem[1:len(elem)-2], "/")

	switch req.Method {
	case http.MethodGet:

		m, ok := m.manifestHandler.Manifest(repo, target)
		if !ok {
			return &regError{
				Status:  http.StatusNotFound,
				Code:    "MANIFEST_UNKNOWN",
				Message: "Unknown manifest",
			}
		}

		h, _, _ := v1.SHA256(bytes.NewReader(m.Blob))
		resp.Header().Set("Docker-Content-Digest", h.String())
		resp.Header().Set("Content-Type", m.ContentType)
		resp.Header().Set("Content-Length", fmt.Sprint(len(m.Blob)))
		resp.WriteHeader(http.StatusOK)
		io.Copy(resp, bytes.NewReader(m.Blob))
		return nil

	case http.MethodHead:

		if ok := m.manifestHandler.RepoExists(repo); !ok {
			return &regError{
				Status:  http.StatusNotFound,
				Code:    "NAME_UNKNOWN",
				Message: "Unknown name",
			}
		}
		m, ok := m.manifestHandler.Manifest(repo, target)
		if !ok {
			return &regError{
				Status:  http.StatusNotFound,
				Code:    "MANIFEST_UNKNOWN",
				Message: "Unknown manifest",
			}
		}

		h, _, _ := v1.SHA256(bytes.NewReader(m.Blob))
		resp.Header().Set("Docker-Content-Digest", h.String())
		resp.Header().Set("Content-Type", m.ContentType)
		resp.Header().Set("Content-Length", fmt.Sprint(len(m.Blob)))
		resp.WriteHeader(http.StatusOK)
		return nil

	case http.MethodPut:
		b := &bytes.Buffer{}
		io.Copy(b, req.Body)
		h, _, _ := v1.SHA256(bytes.NewReader(b.Bytes()))
		digest := h.String()
		mf := Manifest{
			Blob:        b.Bytes(),
			ContentType: req.Header.Get("Content-Type"),
		}

		// If the manifest is a manifest list, check that the manifest
		// list's constituent manifests are already uploaded.
		// This isn't strictly required by the registry API, but some
		// registries require this.
		if types.MediaType(mf.ContentType).IsIndex() {
			if err := func() *regError {

				im, err := v1.ParseIndexManifest(b)
				if err != nil {
					return &regError{
						Status:  http.StatusBadRequest,
						Code:    "MANIFEST_INVALID",
						Message: err.Error(),
					}
				}
				for _, desc := range im.Manifests {
					if !desc.MediaType.IsDistributable() {
						continue
					}
					if desc.MediaType.IsIndex() || desc.MediaType.IsImage() {
						if _, found := m.manifestHandler.Manifest(repo, desc.Digest.String()); !found {
							return &regError{
								Status:  http.StatusNotFound,
								Code:    "MANIFEST_UNKNOWN",
								Message: fmt.Sprintf("Sub-manifest %q not found", desc.Digest),
							}
						}
					} else {
						// TODO: Probably want to do an existence check for blobs.
						m.log.Printf("TODO: Check blobs for %q", desc.Digest)
					}
				}
				return nil
			}(); err != nil {
				return err
			}
		}

		if ok := m.manifestHandler.RepoExists(repo); !ok {
			m.manifestHandler.SetManifests(repo, make(map[string]Manifest, 2))
		}

		// Allow future references by target (tag) and immutable digest.
		// See https://docs.docker.com/engine/reference/commandline/pull/#pull-an-image-by-digest-immutable-identifier.
		m.manifestHandler.SetManifest(repo, digest, mf)
		m.manifestHandler.SetManifest(repo, target, mf)

		if cbh != nil {
			if err := cbh.OnImageReceived(repo, target, digest); err != nil {
				return &regError{
					Status:  http.StatusInternalServerError,
					Code:    "CALLBACK_FAILED",
					Message: err.Error(),
				}
			}
		}
		resp.Header().Set("Docker-Content-Digest", digest)
		resp.WriteHeader(http.StatusCreated)
		return nil

	case http.MethodDelete:
		if ok := m.manifestHandler.RepoExists(repo); !ok {
			return &regError{
				Status:  http.StatusNotFound,
				Code:    "NAME_UNKNOWN",
				Message: "Unknown name",
			}
		}

		_, ok := m.manifestHandler.Manifest(repo, target)
		if !ok {
			return &regError{
				Status:  http.StatusNotFound,
				Code:    "MANIFEST_UNKNOWN",
				Message: "Unknown manifest",
			}
		}

		m.manifestHandler.DeleteManifest(repo, target)
		resp.WriteHeader(http.StatusAccepted)
		return nil

	default:
		return &regError{
			Status:  http.StatusBadRequest,
			Code:    "METHOD_UNKNOWN",
			Message: "We don't understand your method + url",
		}
	}
}

func (m *manifests) handleTags(resp http.ResponseWriter, req *http.Request) *regError {
	elem := strings.Split(req.URL.Path, "/")
	elem = elem[1:]
	repo := strings.Join(elem[1:len(elem)-2], "/")

	if req.Method == "GET" {

		c, ok := m.manifestHandler.Manifests(repo)
		if !ok {
			return &regError{
				Status:  http.StatusNotFound,
				Code:    "NAME_UNKNOWN",
				Message: "Unknown name",
			}
		}

		var tags []string
		for tag := range c {
			if !strings.Contains(tag, "sha256:") {
				tags = append(tags, tag)
			}
		}
		sort.Strings(tags)

		// https://github.com/opencontainers/distribution-spec/blob/b505e9cc53ec499edbd9c1be32298388921bb705/detail.md#tags-paginated
		// Offset using last query parameter.
		if last := req.URL.Query().Get("last"); last != "" {
			for i, t := range tags {
				if t > last {
					tags = tags[i:]
					break
				}
			}
		}

		// Limit using n query parameter.
		if ns := req.URL.Query().Get("n"); ns != "" {
			if n, err := strconv.Atoi(ns); err != nil {
				return &regError{
					Status:  http.StatusBadRequest,
					Code:    "BAD_REQUEST",
					Message: fmt.Sprintf("parsing n: %v", err),
				}
			} else if n < len(tags) {
				tags = tags[:n]
			}
		}

		tagsToList := listTags{
			Name: repo,
			Tags: tags,
		}

		msg, _ := json.Marshal(tagsToList)
		resp.Header().Set("Content-Length", fmt.Sprint(len(msg)))
		resp.WriteHeader(http.StatusOK)
		io.Copy(resp, bytes.NewReader([]byte(msg)))
		return nil
	}

	return &regError{
		Status:  http.StatusBadRequest,
		Code:    "METHOD_UNKNOWN",
		Message: "We don't understand your method + url",
	}
}

func (m *manifests) handleCatalog(resp http.ResponseWriter, req *http.Request) *regError {
	query := req.URL.Query()
	nStr := query.Get("n")
	n := 10000
	if nStr != "" {
		n, _ = strconv.Atoi(nStr)
	}

	if req.Method == "GET" {

		repos := m.manifestHandler.AllRepos()
		if n < len(repos) {
			repos = repos[:n]
		}
		repositoriesToList := catalog{
			Repos: repos,
		}

		msg, _ := json.Marshal(repositoriesToList)
		resp.Header().Set("Content-Length", fmt.Sprint(len(msg)))
		resp.WriteHeader(http.StatusOK)
		io.Copy(resp, bytes.NewReader([]byte(msg)))
		return nil
	}

	return &regError{
		Status:  http.StatusBadRequest,
		Code:    "METHOD_UNKNOWN",
		Message: "We don't understand your method + url",
	}
}

// TODO: implement handling of artifactType querystring
func (m *manifests) handleReferrers(resp http.ResponseWriter, req *http.Request) *regError {
	// Ensure this is a GET request
	if req.Method != "GET" {
		return &regError{
			Status:  http.StatusBadRequest,
			Code:    "METHOD_UNKNOWN",
			Message: "We don't understand your method + url",
		}
	}

	elem := strings.Split(req.URL.Path, "/")
	elem = elem[1:]
	target := elem[len(elem)-1]
	repo := strings.Join(elem[1:len(elem)-2], "/")

	// Validate that incoming target is a valid digest
	if _, err := v1.NewHash(target); err != nil {
		return &regError{
			Status:  http.StatusBadRequest,
			Code:    "UNSUPPORTED",
			Message: "Target must be a valid digest",
		}
	}

	digestToManifestMap, repoExists := m.manifestHandler.Manifests(repo)
	if !repoExists {
		return &regError{
			Status:  http.StatusNotFound,
			Code:    "NAME_UNKNOWN",
			Message: "Unknown name",
		}
	}

	im := v1.IndexManifest{
		SchemaVersion: 2,
		MediaType:     types.OCIImageIndex,
		Manifests:     []v1.Descriptor{},
	}
	for digest, manifest := range digestToManifestMap {
		h, err := v1.NewHash(digest)
		if err != nil {
			continue
		}
		var refPointer struct {
			Subject *v1.Descriptor `json:"subject"`
		}
		json.Unmarshal(manifest.Blob, &refPointer)
		if refPointer.Subject == nil {
			continue
		}
		referenceDigest := refPointer.Subject.Digest
		if referenceDigest.String() != target {
			continue
		}
		// At this point, we know the current digest references the target
		var imageAsArtifact struct {
			Config struct {
				MediaType string `json:"mediaType"`
			} `json:"config"`
		}
		json.Unmarshal(manifest.Blob, &imageAsArtifact)
		im.Manifests = append(im.Manifests, v1.Descriptor{
			MediaType:    types.MediaType(manifest.ContentType),
			Size:         int64(len(manifest.Blob)),
			Digest:       h,
			ArtifactType: imageAsArtifact.Config.MediaType,
		})
	}
	msg, _ := json.Marshal(&im)
	resp.Header().Set("Content-Length", fmt.Sprint(len(msg)))
	resp.Header().Set("Content-Type", string(types.OCIImageIndex))
	resp.WriteHeader(http.StatusOK)
	io.Copy(resp, bytes.NewReader([]byte(msg)))
	return nil
}
