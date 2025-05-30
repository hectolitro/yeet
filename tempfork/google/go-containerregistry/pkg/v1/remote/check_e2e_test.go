// Copyright 2019 Google LLC All Rights Reserved.
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

//go:build integration
// +build integration

package remote

import (
	"net/http"
	"testing"

	"github.com/yeetrun/yeet/tempfork/google/go-containerregistry/pkg/authn"
	"github.com/yeetrun/yeet/tempfork/google/go-containerregistry/pkg/name"
)

func TestCheckPushPermission_Real(t *testing.T) {
	// Tests should not run in an environment where these registries can
	// be pushed to.
	for _, r := range []name.Reference{
		name.MustParseReference("ubuntu"),
		name.MustParseReference("google/cloud-sdk"),
		name.MustParseReference("microsoft/dotnet:sdk"),
		name.MustParseReference("gcr.io/non-existent-project/made-up"),
		name.MustParseReference("gcr.io/google-containers/foo"),
		name.MustParseReference("quay.io/username/reponame"),
	} {
		t.Run(r.String(), func(t *testing.T) {
			t.Parallel()
			if err := CheckPushPermission(r, authn.DefaultKeychain, http.DefaultTransport); err == nil {
				t.Errorf("CheckPushPermission(%s) returned nil", r)
			}
		})
	}
}
