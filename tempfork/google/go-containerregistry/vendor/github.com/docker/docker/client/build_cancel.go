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
	"net/url"
)

// BuildCancel requests the daemon to cancel the ongoing build request.
func (cli *Client) BuildCancel(ctx context.Context, id string) error {
	query := url.Values{}
	query.Set("id", id)

	serverResp, err := cli.post(ctx, "/build/cancel", query, nil, nil)
	ensureReaderClosed(serverResp)
	return err
}
