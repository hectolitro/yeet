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

package random

import (
	"archive/tar"
	"bytes"
	"crypto"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"

	v1 "github.com/yeetrun/yeet/tempfork/google/go-containerregistry/pkg/v1"
	"github.com/yeetrun/yeet/tempfork/google/go-containerregistry/pkg/v1/empty"
	"github.com/yeetrun/yeet/tempfork/google/go-containerregistry/pkg/v1/mutate"
	"github.com/yeetrun/yeet/tempfork/google/go-containerregistry/pkg/v1/partial"
	"github.com/yeetrun/yeet/tempfork/google/go-containerregistry/pkg/v1/types"
)

// uncompressedLayer implements partial.UncompressedLayer from raw bytes.
type uncompressedLayer struct {
	diffID    v1.Hash
	mediaType types.MediaType
	content   []byte
}

// DiffID implements partial.UncompressedLayer
func (ul *uncompressedLayer) DiffID() (v1.Hash, error) {
	return ul.diffID, nil
}

// Uncompressed implements partial.UncompressedLayer
func (ul *uncompressedLayer) Uncompressed() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBuffer(ul.content)), nil
}

// MediaType returns the media type of the layer
func (ul *uncompressedLayer) MediaType() (types.MediaType, error) {
	return ul.mediaType, nil
}

var _ partial.UncompressedLayer = (*uncompressedLayer)(nil)

// Image returns a pseudo-randomly generated Image.
func Image(byteSize, layers int64, options ...Option) (v1.Image, error) {
	adds := make([]mutate.Addendum, 0, 5)
	for i := int64(0); i < layers; i++ {
		layer, err := Layer(byteSize, types.DockerLayer, options...)
		if err != nil {
			return nil, err
		}
		adds = append(adds, mutate.Addendum{
			Layer: layer,
			History: v1.History{
				Author:    "random.Image",
				Comment:   fmt.Sprintf("this is a random history %d of %d", i, layers),
				CreatedBy: "random",
			},
		})
	}

	return mutate.Append(empty.Image, adds...)
}

// Layer returns a layer with pseudo-randomly generated content.
func Layer(byteSize int64, mt types.MediaType, options ...Option) (v1.Layer, error) {
	o := getOptions(options)
	rng := rand.New(o.source) //nolint:gosec

	fileName := fmt.Sprintf("random_file_%d.txt", rng.Int())

	// Hash the contents as we write it out to the buffer.
	var b bytes.Buffer
	hasher := crypto.SHA256.New()
	mw := io.MultiWriter(&b, hasher)

	// Write a single file with a random name and random contents.
	tw := tar.NewWriter(mw)
	if err := tw.WriteHeader(&tar.Header{
		Name:     fileName,
		Size:     byteSize,
		Typeflag: tar.TypeReg,
	}); err != nil {
		return nil, err
	}
	if _, err := io.CopyN(tw, rng, byteSize); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}

	h := v1.Hash{
		Algorithm: "sha256",
		Hex:       hex.EncodeToString(hasher.Sum(make([]byte, 0, hasher.Size()))),
	}

	return partial.UncompressedToLayer(&uncompressedLayer{
		diffID:    h,
		mediaType: mt,
		content:   b.Bytes(),
	})
}
