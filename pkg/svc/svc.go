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

package svc

import (
	"errors"

	"github.com/yeetrun/yeet/pkg/cmdutil"
	"github.com/yeetrun/yeet/pkg/db"
)

var (
	ErrNotInstalled = errors.New("the service is not installed")
)

type LogOptions struct {
	Follow bool
	Lines  int
}

// NewSystemdService creates a new systemd service from a SystemdConfigView.
func NewSystemdService(db *db.Store, cfg db.ServiceView, runDir string) (*SystemdService, error) {
	return &SystemdService{db: db, cfg: cfg, runDir: runDir}, nil
}

// NewDockerComposeService creates a new docker compose service from a config.
func NewDockerComposeService(db *db.Store, cfg db.ServiceView, registryAddr string, images map[db.ImageRepoName]*db.ImageRepo, dataDir, runDir string) (*DockerComposeService, error) {
	sd, err := NewSystemdService(db, cfg, runDir)
	if err != nil {
		return nil, err
	}
	return &DockerComposeService{
		Name:                 cfg.Name(),
		cfg:                  cfg.AsStruct(),
		DataDir:              dataDir,
		NewCmd:               cmdutil.NewStdCmd,
		InternalRegistryAddr: registryAddr,
		Images:               images,
		sd:                   sd,
	}, nil
}
