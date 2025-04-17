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

package swarm // import "github.com/docker/docker/api/types/swarm"

import "os"

// Secret represents a secret.
type Secret struct {
	ID string
	Meta
	Spec SecretSpec
}

// SecretSpec represents a secret specification from a secret in swarm
type SecretSpec struct {
	Annotations
	Data   []byte  `json:",omitempty"`
	Driver *Driver `json:",omitempty"` // name of the secrets driver used to fetch the secret's value from an external secret store

	// Templating controls whether and how to evaluate the secret payload as
	// a template. If it is not set, no templating is used.
	Templating *Driver `json:",omitempty"`
}

// SecretReferenceFileTarget is a file target in a secret reference
type SecretReferenceFileTarget struct {
	Name string
	UID  string
	GID  string
	Mode os.FileMode
}

// SecretReference is a reference to a secret in swarm
type SecretReference struct {
	File       *SecretReferenceFileTarget
	SecretID   string
	SecretName string
}
