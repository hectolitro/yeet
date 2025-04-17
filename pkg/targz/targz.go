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

package targz

import (
	"archive/tar"
	"compress/gzip"
	"io"
)

type Reader struct {
	z *gzip.Reader
	r *tar.Reader
}

func (r Reader) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

func (r Reader) Close() error {
	return r.z.Close()
}

func (r Reader) Next() (*tar.Header, error) {
	return r.r.Next()
}

func New(r io.Reader) (*Reader, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &Reader{z: gz, r: tar.NewReader(gz)}, nil
}

// ReadFile calls f for each entry in the tarball.
func ReadFile(r io.Reader, f func(*tar.Header, io.Reader) error) error {
	t, err := New(r)
	if err != nil {
		return err
	}
	defer t.Close()

	for {
		header, err := t.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err := f(header, t); err != nil {
			return err
		}
	}
	return nil
}
