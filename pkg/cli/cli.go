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

package cli

import (
	"io"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

type CommandHandler struct {
	client io.ReadWriter
	runE   RunE
}

type RunE func(cmd *cobra.Command, args []string) error

func NewCommandHandler(client io.ReadWriter, runE RunE) *CommandHandler {
	return &CommandHandler{client, runE}
}

func (h *CommandHandler) RootCmd(name string) *cobra.Command {
	cmd := &cobra.Command{
		Use: name,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.SetIn(h.client)
	cmd.SetOutput(h.client)

	cmd.AddCommand(
		h.cronCmd(),
		h.disableCmd(),
		h.editCmd(),
		h.envCmd(),
		h.enableCmd(),
		h.eventsCmd(),
		h.logsCmd(),
		h.mountCmd(),
		h.ipCmd(),
		h.umountCmd(),
		h.removeCmd(),
		h.restartCmd(),
		h.rollbackCmd(),
		h.runCmd(),
		h.startCmd(),
		h.stageCmd(),
		h.statusCmd(),
		h.tsCmd(),
		h.stopCmd(),
		h.versionCmd(),
	)

	return cmd
}

// MergeUndefinedFlagsIntoArgs appends all undefined flags from argsIn to args.
// If there are positional arguments after an undefined flag, they are also
// appended to args. Undefined flags are checked against cmd.Flags().Lookup(...)
// to determine if they are defined.
func MergeUndefinedFlagsIntoArgs(argsIn []string, cmd *cobra.Command, args []string) []string {
	// Collect undefined flags and append them to the args list
	appendAllAfter := false
	for _, arg := range argsIn {
		// Fast path if they passed "--" then ignore everything
		if arg == "--" {
			return args
		}
		if strings.HasPrefix(arg, "--") && cmd.Flags().Lookup(strings.TrimPrefix(strings.SplitN(arg, "=", 2)[0], "--")) == nil {
			// If it's an undefined flag, append it to args
			args = append(args, arg)
			appendAllAfter = true
		} else if appendAllAfter {
			// If we've seen an undefined flag, append all following args
			args = append(args, arg)
		}
	}
	return args
}

func (h *CommandHandler) envCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "env",
		Short: "Manage environment variables",
		RunE:  h.runE,
	}
	return c
}

// VersionCommit returns the commit hash of the current build.
func VersionCommit() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	var dirty bool
	var commit string
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			commit = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if commit == "" {
		return "dev"
	}

	if len(commit) >= 9 {
		commit = commit[:9]
	}
	if dirty {
		commit += "+dirty"
	}
	return commit
}

func (h *CommandHandler) versionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "version",
		Short: "Show the version of the Catch server",
		RunE:  h.runE,
	}
	c.Flags().Bool("json", false, "Output as JSON")
	return c
}

func (h *CommandHandler) stageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stage",
		Short: "Stage a service",
		RunE:  h.runE,
		// Relax the flag parsing to allow unknown flags to be set on the
		// service config.
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
	}
	cmd.Flags().String("net", "", "Network to connect to")
	cmd.Flags().String("ts-ver", "", "Tailscale version to use; when net=ts")
	cmd.Flags().String("ts-exit", "", "Tailscale exit node to use; when net=ts")
	cmd.Flags().StringArray("ts-tags", nil, "Tailscale tags to use; when net=ts")
	cmd.Flags().String("ts-auth-key", "", "Tailscale auth key to use; when net=ts")
	cmd.Flags().String("macvlan-mac", "", "Macvlan interface mac address to use; when net=macvlan")
	cmd.Flags().Int("macvlan-vlan", 0, "Macvlan VLAN ID to use; when net=macvlan")
	cmd.Flags().String("macvlan-parent", "", "Macvlan parent interface; when net=macvlan")

	show := &cobra.Command{
		Use:   "show",
		Short: "Show the staged configuration",
		RunE:  h.runE,
	}
	show.PersistentFlags().Bool("env", false, "Show environment variables")
	cmd.AddCommand(show)
	cmd.AddCommand(&cobra.Command{
		Use:   "clear",
		Short: "Clear the staged configuration",
		RunE:  h.runE,
	})

	commit := &cobra.Command{
		Use:   "commit",
		Short: "Commit the staged configuration",
		RunE:  h.runE,
	}
	commit.PersistentFlags().Bool("restart", true, "Whether to restart the service after committing")
	cmd.AddCommand(commit)
	return cmd
}

func (h *CommandHandler) runCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Install a service with the binary received from stdin",
		RunE:  h.runE,
		// Relax the flag parsing to allow unknown flags to be set on the
		// service config.
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
	}
	cmd.Flags().String("net", "", "Network to connect to")
	cmd.Flags().String("ts-ver", "", "Tailscale version to use; when net=ts")
	cmd.Flags().String("ts-exit", "", "Tailscale exit node to use; when net=ts")
	cmd.Flags().StringArray("ts-tags", nil, "Tailscale tags to use; when net=ts")
	cmd.Flags().String("ts-auth-key", "", "Tailscale auth key to use; when net=ts")
	cmd.Flags().String("macvlan-mac", "", "Macvlan interface mac address to use; when net=macvlan")
	cmd.Flags().Int("macvlan-vlan", 0, "Macvlan VLAN ID to use; when net=macvlan")
	cmd.Flags().String("macvlan-parent", "", "Macvlan parent interface; when net=macvlan")
	cmd.Flags().Bool("restart", true, "Whether to restart the service after installation")

	return cmd
}

func (h *CommandHandler) startCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start a service",
		RunE:  h.runE,
	}
}

func (h *CommandHandler) stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop a service",
		RunE:  h.runE,
	}
}

func (h *CommandHandler) rollbackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rollback",
		Short: "Rollback a service",
		RunE:  h.runE,
	}
}

func (h *CommandHandler) restartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart a service",
		RunE:  h.runE,
	}
}

func (h *CommandHandler) editCmd() *cobra.Command {
	edit := &cobra.Command{
		Use:   "edit",
		Short: "Edit a service",
		RunE:  h.runE,
	}
	edit.PersistentFlags().Bool("env", false, "Edit environment variables")
	edit.PersistentFlags().Bool("config", false, "Edit internal configuration")
	edit.PersistentFlags().Bool("ts", false, "Edit Tailscale configuration")
	// TODO: We have to add this flag otherwise restart=false which is not what we want
	edit.PersistentFlags().Bool("restart", true, "Whether to restart the service after editing")
	return edit
}

func (h *CommandHandler) enableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable a service",
		RunE:  h.runE,
	}
}

func (h *CommandHandler) disableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable a service",
		RunE:  h.runE,
	}
}

func (h *CommandHandler) logsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Show logs of a service",
		RunE:  h.runE,
	}
	cmd.Flags().BoolP("follow", "f", false, "Follow the logs")
	cmd.Flags().IntP("lines", "n", -1, "Number of lines to show from the end of the logs")
	return cmd
}

func (h *CommandHandler) tsCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "ts",
		Short:              "Run a tailscale command",
		RunE:               h.runE,
		DisableFlagParsing: true,
	}
}

func (h *CommandHandler) statusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of a service",
		RunE:  h.runE,
	}
	cmd.Flags().String("format", "table", "Output format (table, json, json-pretty)")
	return cmd
}

func (h *CommandHandler) cronCmd() *cobra.Command {
	return &cobra.Command{
		Use:   `cron "<cron expression>" [-- <binary args>]`,
		Short: "Install a cron with the binary received from stdin",
		Args:  cobra.MinimumNArgs(2),
		RunE:  h.runE,
	}
}

func (h *CommandHandler) removeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Remove a service",
		RunE:  h.runE,
	}
}

func (h *CommandHandler) eventsCmd() *cobra.Command {
	events := &cobra.Command{
		Use:   "events",
		Short: "Show events for a service",
		RunE:  h.runE,
	}
	events.Flags().Bool("all", false, "Show all events")
	return events
}

func (h *CommandHandler) mountCmd() *cobra.Command {
	mountCmd := &cobra.Command{
		Use:   "mount | host:path [target] [--type=nfs] [--opts=default]",
		Short: "Mount a directory from a host",
		RunE:  h.runE,
	}
	mountCmd.Flags().StringP("type", "t", "nfs", "Type of mount (e.g., nfs)")
	mountCmd.Flags().StringP("opts", "o", "defaults", "Mount options")
	mountCmd.Flags().StringSlice("deps", nil, "Dependencies expressed as a comma separated list of unit names")
	return mountCmd
}

func (h *CommandHandler) ipCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ip",
		Short: "Show the IP addresses of a service",
		RunE:  h.runE,
	}
}

func (h *CommandHandler) umountCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "umount",
		Short: "Unmount a directory",
		RunE:  h.runE,
	}
}
