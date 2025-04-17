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

package client // import "github.com/docker/docker/client"

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/docker/docker/api/types/registry"
)

// DistributionInspect returns the image digest with the full manifest.
func (cli *Client) DistributionInspect(ctx context.Context, image, encodedRegistryAuth string) (registry.DistributionInspect, error) {
	// Contact the registry to retrieve digest and platform information
	var distributionInspect registry.DistributionInspect
	if image == "" {
		return distributionInspect, objectNotFoundError{object: "distribution", id: image}
	}

	if err := cli.NewVersionError("1.30", "distribution inspect"); err != nil {
		return distributionInspect, err
	}
	var headers map[string][]string

	if encodedRegistryAuth != "" {
		headers = map[string][]string{
			registry.AuthHeader: {encodedRegistryAuth},
		}
	}

	resp, err := cli.get(ctx, "/distribution/"+image+"/json", url.Values{}, headers)
	defer ensureReaderClosed(resp)
	if err != nil {
		return distributionInspect, err
	}

	err = json.NewDecoder(resp.body).Decode(&distributionInspect)
	return distributionInspect, err
}
