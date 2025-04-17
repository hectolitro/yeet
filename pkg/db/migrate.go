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

package db

import (
	"fmt"
	"log"

	"tailscale.com/util/mak"
)

const CurrentDataVersion = 5

var migrators = map[int]func(*Data) error{ // Start DataVersion -> NextStep
	3: reinit,
	4: addDockerEndpoints,
}

func reinit(d *Data) error {
	log.Fatal("Migration required but not supported, please delete the db file")
	return fmt.Errorf("unreachable")
}

func addDockerEndpoints(d *Data) error {
	for _, net := range d.DockerNetworks {
		for k, ep := range net.EndpointAddrs {
			mak.Set(&net.Endpoints, k, &DockerEndpoint{
				EndpointID: k,
				IPv4:       ep,
			})
		}
		net.EndpointAddrs = nil
	}
	return nil
}
