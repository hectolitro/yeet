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
	"io"
	"net/url"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/registry"
	"github.com/pkg/errors"
)

// PluginUpgrade upgrades a plugin
func (cli *Client) PluginUpgrade(ctx context.Context, name string, options types.PluginInstallOptions) (rc io.ReadCloser, err error) {
	if err := cli.NewVersionError("1.26", "plugin upgrade"); err != nil {
		return nil, err
	}
	query := url.Values{}
	if _, err := reference.ParseNormalizedNamed(options.RemoteRef); err != nil {
		return nil, errors.Wrap(err, "invalid remote reference")
	}
	query.Set("remote", options.RemoteRef)

	privileges, err := cli.checkPluginPermissions(ctx, query, options)
	if err != nil {
		return nil, err
	}

	resp, err := cli.tryPluginUpgrade(ctx, query, privileges, name, options.RegistryAuth)
	if err != nil {
		return nil, err
	}
	return resp.body, nil
}

func (cli *Client) tryPluginUpgrade(ctx context.Context, query url.Values, privileges types.PluginPrivileges, name, registryAuth string) (serverResponse, error) {
	headers := map[string][]string{registry.AuthHeader: {registryAuth}}
	return cli.post(ctx, "/plugins/"+name+"/upgrade", query, privileges, headers)
}
