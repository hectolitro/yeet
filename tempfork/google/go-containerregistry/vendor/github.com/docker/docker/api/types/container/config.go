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

package container // import "github.com/docker/docker/api/types/container"

import (
	"io"
	"time"

	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
)

// MinimumDuration puts a minimum on user configured duration.
// This is to prevent API error on time unit. For example, API may
// set 3 as healthcheck interval with intention of 3 seconds, but
// Docker interprets it as 3 nanoseconds.
const MinimumDuration = 1 * time.Millisecond

// StopOptions holds the options to stop or restart a container.
type StopOptions struct {
	// Signal (optional) is the signal to send to the container to (gracefully)
	// stop it before forcibly terminating the container with SIGKILL after the
	// timeout expires. If not value is set, the default (SIGTERM) is used.
	Signal string `json:",omitempty"`

	// Timeout (optional) is the timeout (in seconds) to wait for the container
	// to stop gracefully before forcibly terminating it with SIGKILL.
	//
	// - Use nil to use the default timeout (10 seconds).
	// - Use '-1' to wait indefinitely.
	// - Use '0' to not wait for the container to exit gracefully, and
	//   immediately proceeds to forcibly terminating the container.
	// - Other positive values are used as timeout (in seconds).
	Timeout *int `json:",omitempty"`
}

// HealthConfig holds configuration settings for the HEALTHCHECK feature.
type HealthConfig struct {
	// Test is the test to perform to check that the container is healthy.
	// An empty slice means to inherit the default.
	// The options are:
	// {} : inherit healthcheck
	// {"NONE"} : disable healthcheck
	// {"CMD", args...} : exec arguments directly
	// {"CMD-SHELL", command} : run command with system's default shell
	Test []string `json:",omitempty"`

	// Zero means to inherit. Durations are expressed as integer nanoseconds.
	Interval    time.Duration `json:",omitempty"` // Interval is the time to wait between checks.
	Timeout     time.Duration `json:",omitempty"` // Timeout is the time to wait before considering the check to have hung.
	StartPeriod time.Duration `json:",omitempty"` // The start period for the container to initialize before the retries starts to count down.

	// Retries is the number of consecutive failures needed to consider a container as unhealthy.
	// Zero means inherit.
	Retries int `json:",omitempty"`
}

// ExecStartOptions holds the options to start container's exec.
type ExecStartOptions struct {
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	ConsoleSize *[2]uint `json:",omitempty"`
}

// Config contains the configuration data about a container.
// It should hold only portable information about the container.
// Here, "portable" means "independent from the host we are running on".
// Non-portable information *should* appear in HostConfig.
// All fields added to this struct must be marked `omitempty` to keep getting
// predictable hashes from the old `v1Compatibility` configuration.
type Config struct {
	Hostname        string              // Hostname
	Domainname      string              // Domainname
	User            string              // User that will run the command(s) inside the container, also support user:group
	AttachStdin     bool                // Attach the standard input, makes possible user interaction
	AttachStdout    bool                // Attach the standard output
	AttachStderr    bool                // Attach the standard error
	ExposedPorts    nat.PortSet         `json:",omitempty"` // List of exposed ports
	Tty             bool                // Attach standard streams to a tty, including stdin if it is not closed.
	OpenStdin       bool                // Open stdin
	StdinOnce       bool                // If true, close stdin after the 1 attached client disconnects.
	Env             []string            // List of environment variable to set in the container
	Cmd             strslice.StrSlice   // Command to run when starting the container
	Healthcheck     *HealthConfig       `json:",omitempty"` // Healthcheck describes how to check the container is healthy
	ArgsEscaped     bool                `json:",omitempty"` // True if command is already escaped (meaning treat as a command line) (Windows specific).
	Image           string              // Name of the image as it was passed by the operator (e.g. could be symbolic)
	Volumes         map[string]struct{} // List of volumes (mounts) used for the container
	WorkingDir      string              // Current directory (PWD) in the command will be launched
	Entrypoint      strslice.StrSlice   // Entrypoint to run when starting the container
	NetworkDisabled bool                `json:",omitempty"` // Is network disabled
	MacAddress      string              `json:",omitempty"` // Mac Address of the container
	OnBuild         []string            // ONBUILD metadata that were defined on the image Dockerfile
	Labels          map[string]string   // List of labels set to this container
	StopSignal      string              `json:",omitempty"` // Signal to stop a container
	StopTimeout     *int                `json:",omitempty"` // Timeout (in seconds) to stop a container
	Shell           strslice.StrSlice   `json:",omitempty"` // Shell for shell-form of RUN, CMD, ENTRYPOINT
}
