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

// RuntimeType is the type of runtime used for the TaskSpec
type RuntimeType string

// RuntimeURL is the proto type url
type RuntimeURL string

const (
	// RuntimeContainer is the container based runtime
	RuntimeContainer RuntimeType = "container"
	// RuntimePlugin is the plugin based runtime
	RuntimePlugin RuntimeType = "plugin"
	// RuntimeNetworkAttachment is the network attachment runtime
	RuntimeNetworkAttachment RuntimeType = "attachment"

	// RuntimeURLContainer is the proto url for the container type
	RuntimeURLContainer RuntimeURL = "types.docker.com/RuntimeContainer"
	// RuntimeURLPlugin is the proto url for the plugin type
	RuntimeURLPlugin RuntimeURL = "types.docker.com/RuntimePlugin"
)

// NetworkAttachmentSpec represents the runtime spec type for network
// attachment tasks
type NetworkAttachmentSpec struct {
	ContainerID string
}
