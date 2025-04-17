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
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yeetrun/yeet/pkg/db"
	"github.com/yeetrun/yeet/pkg/fileutil"
	"tailscale.com/types/lazy"
)

const dockerContainerNamePrefix = "catch"

type DockerComposeStatus map[string]Status

var ErrDockerStatusUnknown = fmt.Errorf("unknown docker status")

var ErrDockerNotFound = fmt.Errorf("docker not found")

type DockerComposeService struct {
	Name                 string
	cfg                  *db.Service
	DataDir              string
	NewCmd               func(name string, arg ...string) *exec.Cmd
	Images               map[db.ImageRepoName]*db.ImageRepo
	InternalRegistryAddr string
	sd                   *SystemdService

	installEnvOnce lazy.SyncValue[error]
}

func do(f ...func() error) error {
	for _, fn := range f {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

// DockerCmd returns the path to the docker binary.
func DockerCmd() (string, error) {
	p, err := exec.LookPath("docker")
	if err != nil {
		return "", ErrDockerNotFound
	}
	return p, nil
}

func (s *DockerComposeService) command(args ...string) (*exec.Cmd, error) {
	dockerPath, err := DockerCmd()
	if err != nil {
		return nil, err
	}
	nargs := []string{
		"compose",
		"--project-name", s.projectName(s.Name),
		"--project-directory", s.DataDir,
	}
	cf, ok := s.cfg.Artifacts.Gen(db.ArtifactDockerComposeFile, s.cfg.Generation)
	if !ok {
		return nil, fmt.Errorf("compose file not found")
	}
	nargs = append(nargs,
		"--file", cf,
	)
	if cf, ok := s.cfg.Artifacts.Gen(db.ArtifactDockerComposeNetwork, s.cfg.Generation); ok {
		nargs = append(nargs, "--file", cf)
	}

	if err := s.installEnvOnce.Get(func() error {
		if ef, ok := s.cfg.Artifacts.Gen(db.ArtifactEnvFile, s.cfg.Generation); ok {
			return fileutil.CopyFile(ef, filepath.Join(s.DataDir, ".env"))
		}
		os.Remove(filepath.Join(s.DataDir, ".env"))
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to copy env file: %v", err)
	}
	args = append(nargs, args...)
	c := s.NewCmd(dockerPath, args...)
	c.Dir = s.DataDir
	return c, nil
}

func (s *DockerComposeService) runCommand(args ...string) error {
	cmd, err := s.command(args...)
	if err != nil {
		return fmt.Errorf("failed to create docker-compose command: %v", err)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run docker command: %v", err)
	}
	return nil
}

func matchingRefs(refs map[db.ImageRepoName]*db.ImageRepo, svcName string, ref db.ImageRef) (matching []string) {
	for rn, ir := range refs {
		if s, _, _ := strings.Cut(string(rn), "/"); s == svcName {
			if _, ok := ir.Refs[ref]; ok {
				matching = append(matching, string(rn))
			}
		}
	}
	return matching
}

// InternalRegistryHost is the domain name for the internal registry.
const InternalRegistryHost = "catchit.dev"

func (s *DockerComposeService) Install() error {
	if err := s.Down(); err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}
	return s.sd.Install()
}

func (s *DockerComposeService) Up() error {
	s.sd.Start()
	// Ok so this is a bit of a hack. We want to use a nice looking image
	// name catchit.dev/svc/img instead of a weirdo loopback
	// 127.0.0.1:42353/svc/img address or with a random port. So to pull
	// this off we first pull the image from the internal registry with the
	// random address, then retag it with the nice looking address, then
	// remove the image with the random address. This is all a bit of a hack
	// but it works for now. We likely want to replace docker with
	// containerd but we need to figure out how to get the same compose
	// functionality with containerd.
	isInternal := false
	for _, ref := range matchingRefs(s.Images, s.Name, "latest") {
		isInternal = true
		internalRef := fmt.Sprintf("%s/%s:latest", s.InternalRegistryAddr, ref)
		canonicalRef := fmt.Sprintf("%s/%s:latest", InternalRegistryHost, ref)
		if err := do(
			s.NewCmd("docker", "pull", internalRef).Run,
			s.NewCmd("docker", "tag", internalRef, canonicalRef).Run,
			s.NewCmd("docker", "rmi", internalRef).Run,
		); err != nil {
			log.Printf("docker tag: %v", err)
			return fmt.Errorf("failed to tag image: %v", err)
		}
	}
	pull := "always"
	if isInternal {
		// Skip pulling from catchit.dev since it's a virtual registry that doesn't actually exist
		pull = "never"
	}
	return s.runCommand("up", "--pull", pull, "-d")
}

func (s *DockerComposeService) Remove() error {
	if err := s.Down(); err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}
	s.sd.Stop()
	return s.sd.Uninstall()
}

func (s *DockerComposeService) Down() error {
	if ok, err := s.Exists(); err != nil {
		return fmt.Errorf("failed to check if service exists: %v", err)
	} else if !ok {
		return nil
	}
	return s.runCommand("down", "--remove-orphans")
}

func (s *DockerComposeService) Start() error {
	s.sd.Start()
	return s.runCommand("start")
}

func (s *DockerComposeService) Stop() error {
	if ok, err := s.Exists(); err != nil {
		return fmt.Errorf("failed to check if service exists: %v", err)
	} else if !ok {
		return nil
	}
	s.sd.Stop()
	return s.runCommand("stop")
}

func (s *DockerComposeService) Restart() error {
	if ok, err := s.Exists(); err != nil {
		return fmt.Errorf("failed to check if service exists: %v", err)
	} else if !ok {
		return nil
	}
	return s.runCommand("restart")
}

func (s *DockerComposeService) Exists() (bool, error) {
	statuses, err := s.Statuses()
	if err != nil {
		if err == ErrDockerStatusUnknown {
			return false, nil
		}
		return false, err
	}
	return len(statuses) > 0, nil
}

func (s *DockerComposeService) Status() (Status, error) {
	return StatusUnknown, fmt.Errorf("not implemented")
}

func (s *DockerComposeService) Statuses() (DockerComposeStatus, error) {
	cmd, err := s.command("ps", "-a",
		"--format", `{{.Label "com.docker.compose.service"}},{{.State}}`)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker-compose command: %v", err)
	}
	cmd.Stdout = nil
	ob, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run docker command: %v (%s)", err, ob)
	}

	output := string(ob)
	if strings.TrimSpace(output) == "" {
		return nil, ErrDockerStatusUnknown
	}

	statuses := make(DockerComposeStatus)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) != 2 {
			log.Printf("unexpected docker-compose ps output: %s", line)
			continue
		}
		cn := fields[0]
		switch fields[1] {
		case "running":
			statuses[cn] = StatusRunning
		case "exited":
			statuses[cn] = StatusStopped
		default:
			statuses[cn] = StatusUnknown
		}
	}
	return statuses, nil
}

func (s *DockerComposeService) Logs(opts *LogOptions) error {
	if opts == nil {
		opts = &LogOptions{}
	}
	args := []string{"logs"}
	if opts.Follow {
		args = append(args, "--follow")
	}
	if opts.Lines > 0 {
		args = append(args, "--tail", strconv.Itoa(opts.Lines))
	}
	return s.runCommand(args...)
}

// projectName returns the docker-compose project name for the given service name.
func (s *DockerComposeService) projectName(sn string) string {
	return fmt.Sprintf("%s-%s", dockerContainerNamePrefix, sn)
}
