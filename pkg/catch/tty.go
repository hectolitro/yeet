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

package catch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/yeetrun/yeet/pkg/cli"
	"github.com/yeetrun/yeet/pkg/cmdutil"
	"github.com/yeetrun/yeet/pkg/cronutil"
	"github.com/yeetrun/yeet/pkg/db"
	"github.com/yeetrun/yeet/pkg/fileutil"
	"github.com/yeetrun/yeet/pkg/svc"
	"github.com/creack/pty"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	gssh "tailscale.com/tempfork/gliderlabs/ssh"
	"tailscale.com/util/mak"
)

const (
	editUnitsSeparator = "=====================================|%s|====================================="
)

var (
	editUnitsSeparatorRe = regexp.MustCompile(`=====================================\|([^|]+)\|=====================================`)
)

type writeCloser interface {
	CloseWrite() error
}

type ttyExecer struct {
	// Inputs
	ctx       context.Context
	args      []string
	s         *Server
	sn        string
	user      string
	rawRW     io.ReadWriter
	rawCloser io.Closer
	isPty     bool
	ptyReq    gssh.Pty
	ptyWCh    <-chan gssh.Window

	// Assigned during run
	rw io.ReadWriter // May be a pty
}

func (e *ttyExecer) run() error {
	var doneWritingToSession chan struct{}
	e.rw = e.rawRW
	var closer io.Closer
	if e.isPty {
		stdin, tty, err := pty.Open()
		if err != nil {
			fmt.Fprintf(e.rw, "Error: %v\n", err)
			return err
		}
		dup, err := syscall.Dup(int(stdin.Fd()))
		if err != nil {
			stdin.Close()
			tty.Close()
			log.Printf("Error duping pty: %v", err)
			return err
		}
		stdout := os.NewFile(uintptr(dup), stdin.Name())

		e.rw = tty
		closer = tty

		setWinsize(tty, e.ptyReq.Window.Width, e.ptyReq.Window.Height)
		if e.ptyWCh != nil {
			go func() {
				for win := range e.ptyWCh {
					setWinsize(tty, win.Width, win.Height)
				}
			}()
		}

		doneWritingToSession = make(chan struct{})
		go func() {
			if c, ok := e.rawRW.(writeCloser); ok {
				defer c.CloseWrite()
			}
			defer stdout.Close()
			defer close(doneWritingToSession)
			if _, err := io.Copy(e.rawRW, stdout); err != nil {
				log.Printf("Error copying from stdout to session: %v", err)
			}
		}()
		go func() {
			defer stdin.Close()
			if _, err := io.Copy(stdin, e.rawRW); err != nil {
				log.Printf("Error copying from session to stdin: %v", err)
			}
		}()
	}

	err := e.exec()
	if err != nil {
		fmt.Fprintf(e.rawRW, "Error: %v\n", err)
	}
	if closer != nil {
		closer.Close()
	}
	if doneWritingToSession != nil {
		<-doneWritingToSession
	}
	return err
}

func (e *ttyExecer) ResizeTTY(cols, rows int) {
	if !e.isPty {
		return
	}
	if tty, ok := e.rw.(*os.File); ok {
		setWinsize(tty, cols, rows)
	}
}

func (e *ttyExecer) exec() error {
	ch := cli.NewCommandHandler(e.rw, e.runE)
	cmd := ch.RootCmd("catch")
	if e.args == nil {
		// If no args are provided, set an empty slice. Otherwise, the cobra will
		// try to parse the os.Args.
		cmd.SetArgs([]string{})
	} else {
		cmd.SetArgs(e.args)
	}
	return cmd.ExecuteContext(e.ctx)
}

func (e *ttyExecer) runE(cmd *cobra.Command, args []string) error {
	subCmdCalledAs := cmd.CalledAs()
	c := cmd
	for c.Parent() != c.Root() && c.Parent() != nil {
		c = c.Parent()
		subCmdCalledAs = c.Use
	}

	switch subCmdCalledAs {
	case "cron":
		cronexpr := strings.Join(args[0:5], " ")
		return e.cronCmdFunc(cmd, cronexpr, args[5:])
	case "disable":
		return e.disableCmdFunc(cmd, args)
	case "edit":
		return e.editCmdFunc(cmd, args)
	case "events":
		return e.eventsCmdFunc(cmd, args)
	case "enable":
		return e.enableCmdFunc(cmd, args)
	case "mount":
		return e.mountCmdFunc(cmd, args)
	case "ip":
		return e.ipCmdFunc(cmd, args)
	case "ts":
		return e.tsCmdFunc(cmd, args)
	case "umount":
		return e.umountCmdFunc(cmd, args)
	case "env":
		return e.envCmdFunc(cmd, args)
	case "logs":
		return e.logsCmdFunc(cmd, args)
	case "remove":
		return e.removeCmdFunc(cmd, args)
	case "restart":
		return e.restartCmdFunc(cmd, args)
	case "rollback":
		return e.rollbackCmdFunc(cmd, args)
	case "run":
		return e.runCmdFunc(cmd, args)
	case "stage":
		return e.stageCmdFunc(cmd, args)
	case "start":
		return e.startCmdFunc(cmd, args)
	case "status":
		return e.statusCmdFunc(cmd, args)
	case "stop":
		return e.stopCmdFunc(cmd, args)
	case "version":
		j, _ := cmd.Flags().GetBool("json")
		if j {
			json.NewEncoder(e.rw).Encode(GetInfo())
		} else {
			fmt.Fprintln(e.rw, VersionCommit())
		}
	default:
		log.Printf("Unhandled command %q", subCmdCalledAs)
		return fmt.Errorf("unhandled command %q", subCmdCalledAs)
	}
	return nil
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

// Human-readable format function
func humanReadableBytes(bts float64) string {
	const unit = 1024
	if bts <= unit {
		return fmt.Sprintf("%.2f B", bts)
	}
	const prefix = "KMGTPE"
	n := bts
	i := -1
	for n > unit {
		i++
		n = n / unit
	}

	return fmt.Sprintf("%.2f %cB", n, prefix[i])
}

// Function to generate spaces to clear old characters
func makePadding(oldLen, newLen int) string {
	if oldLen > newLen {
		return strings.Repeat(" ", oldLen-newLen)
	}
	return ""
}

// install installs a service by reading the binary from the `in` input stream.
// The service is configured via `cfg`, an InstallerCfg struct. Client output
// can be written to `out`. An error is returned if the installation fails.
func (e *ttyExecer) install(in io.Reader, cfg FileInstallerCfg) error {
	if runtime.GOOS == "darwin" {
		// Don't do anything on macOS yet.
		return nil
	}
	e.printf("Installing service %q\n", e.sn)

	inst, err := NewFileInstaller(e.s, cfg)
	if err != nil {
		return fmt.Errorf("failed to create installer: %w", err)
	}
	defer inst.Close()

	if !cfg.EnvFile {
		// Start a goroutine to close the session if no data is received after 1
		// second but only if it's not an env file which can be empty.
		started := make(chan struct{})
		done := make(chan struct{})
		defer close(done)
		go func() {
			select {
			case <-e.ctx.Done():
				return
			case <-started:
			case <-done:
				return
			case <-time.After(time.Second):
				e.printf("Error: timeout waiting for bytes in\n")
				if e.rawCloser != nil {
					e.rawCloser.Close()
				}
				return
			}

			// Keep track of the longest printed string length
			var lastPrintedLen int

			print := func() {
				humanReadable := fmt.Sprintf("\rReceived: %s\tRate: %s/s", humanReadableBytes(inst.Received()), humanReadableBytes(inst.Rate()))
				ln := len(humanReadable)
				e.printf("%s%s", humanReadable, makePadding(lastPrintedLen, ln))

				lastPrintedLen = ln
			}

			for {
				select {
				case <-e.ctx.Done():
					return
				case <-done:
					print()
					e.printf("\n")
					return
				case <-time.After(100 * time.Millisecond):
					print()
				}
			}
		}()
		if _, err := io.CopyN(inst, in, 1); err != nil {
			inst.failed = true
			e.printf("Error: failed to read binary\n")
			return fmt.Errorf("failed to read binary: %w", err)
		}
		log.Print("Started receiving binary")
		close(started)
	}

	// Now copy the rest of the file
	if _, err := io.Copy(inst, in); err != nil {
		inst.failed = true
		e.printf("Error: failed to read binary: %v\n", err)
		return fmt.Errorf("failed to copy to installer: %w", err)
	}
	return nil
}

func (e *ttyExecer) printf(format string, a ...any) {
	fmt.Fprintf(e.rw, format, a...)
}

func (e *ttyExecer) fileInstaller(cmd *cobra.Command, argsIn []string) FileInstallerCfg {
	var args []string
	if len(argsIn) > 0 {
		args = argsIn
	}
	ic := e.installerCfg()
	return FileInstallerCfg{
		InstallerCfg: ic,
		Network: NetworkOpts{
			Interfaces: First(cmd.Flags().GetString("net")),
			Tailscale: TailscaleOpts{
				Version:  First(cmd.Flags().GetString("ts-ver")),
				Tags:     First(cmd.Flags().GetStringArray("ts-tags")),
				ExitNode: First(cmd.Flags().GetString("ts-exit")),
				AuthKey:  First(cmd.Flags().GetString("ts-auth-key")),
			},
			Macvlan: MacvlanOpts{
				Parent: First(cmd.Flags().GetString("macvlan-parent")),
				Mac:    First(cmd.Flags().GetString("macvlan-mac")),
				VLAN:   First(cmd.Flags().GetInt("macvlan-vlan")),
			},
		},
		Args:   args,
		NewCmd: e.newCmd,
	}
}

func (e *ttyExecer) installerCfg() InstallerCfg {
	return InstallerCfg{
		ServiceName:      e.sn,
		User:             e.user,
		Printer:          e.printf,
		ClientOut:        e.rw,
		SSHSessionCloser: sessionCloser{e.rawCloser},
	}
}

func (e *ttyExecer) runCmdFunc(cmd *cobra.Command, argsIn []string) error {
	if e.sn == SystemService {
		return fmt.Errorf("cannot %s, reserved service name", cmd.CalledAs())
	}
	cfg := e.fileInstaller(cmd, argsIn)
	return e.install(e.rw, cfg)
}

type sessionCloser struct {
	io.Closer
}

func (s sessionCloser) Close() error {
	if s.Closer != nil {
		// If the closer is a gssh.Session, call Exit(0) on it.
		if closer, ok := s.Closer.(gssh.Session); ok {
			closer.Exit(0)
		}
	}
	return nil
}

func (e *ttyExecer) stageCmdFunc(cmd *cobra.Command, args []string) error {
	// Merge any undefined flags into the args slice.
	args = cli.MergeUndefinedFlagsIntoArgs(e.args, cmd, args)

	if e.sn == SystemService {
		return fmt.Errorf("cannot stage system service")
	}
	fi := e.fileInstaller(cmd, args)
	if err := e.s.ensureDirs(e.sn, e.user); err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}
	fi.NoBinary = true
	switch cmd.CalledAs() {
	case "show":
		sv, err := e.s.serviceView(e.sn)
		if err != nil {
			log.Printf("%v", err)
		}
		if showEnv, _ := cmd.PersistentFlags().GetBool("env"); showEnv {
			if err := e.s.printEnv(e.rw, sv, true); err != nil {
				return fmt.Errorf("failed to print env: %w", err)
			}
		} else {
			fmt.Fprintf(e.rw, "%s\n", asJSON(sv))
		}
	case "clear":
		return fmt.Errorf("not implemented")
	case "stage", "commit":
		fi.StageOnly = cmd.CalledAs() == "stage"
		inst, err := NewFileInstaller(e.s, fi)
		if err != nil {
			return fmt.Errorf("failed to create installer: %w", err)
		}
		if err := inst.Close(); err != nil {
			return fmt.Errorf("failed to close installer: %w", err)
		}
		sv, err := e.s.serviceView(e.sn)
		if err != nil {
			log.Printf("%v", err)
		}
		if fi.StageOnly {
			fmt.Fprintf(e.rw, "%s\n", asJSON(sv))
		}
	default:
		return fmt.Errorf("invalid argument %q", cmd.CalledAs())
	}
	return nil
}

func (e *ttyExecer) startCmdFunc(_ *cobra.Command, _ []string) error {
	if e.sn == SystemService || e.sn == CatchService {
		return fmt.Errorf("cannot start system service")
	}
	runner, err := e.serviceRunner()
	if err != nil {
		return fmt.Errorf("failed to get service runner: %w", err)
	}
	if err := runner.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	return nil
}

func (e *ttyExecer) stopCmdFunc(_ *cobra.Command, _ []string) error {
	if e.sn == SystemService || e.sn == CatchService {
		return fmt.Errorf("cannot stop system service")
	}
	runner, err := e.serviceRunner()
	if err != nil {
		return fmt.Errorf("failed to get service runner: %w", err)
	}
	if err := runner.Stop(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	return nil
}

func (e *ttyExecer) rollbackCmdFunc(cmd *cobra.Command, _ []string) error {
	_, s, err := e.s.cfg.DB.MutateService(e.sn, func(d *db.Data, s *db.Service) error {
		if s.Generation == 0 {
			return fmt.Errorf("no generation to rollback")
		}
		minG := s.LatestGeneration - maxGenerations
		gen := s.Generation - 1
		if gen < minG {
			return fmt.Errorf("generation %d is too old, earliest rollback is %d", gen, minG)
		}
		if gen == 0 {
			return fmt.Errorf("generation %d is the oldest, cannot rollback", s.Generation)
		}
		s.Generation = gen
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to rollback service: %w", err)
	}
	cfg := e.installerCfg()
	i, err := e.s.NewInstaller(cfg)
	if err != nil {
		return fmt.Errorf("failed to create installer: %w", err)
	}
	i.NewCmd = e.newCmd
	return i.InstallGen(s.Generation)
}

func (e *ttyExecer) restartCmdFunc(_ *cobra.Command, _ []string) error {
	e.printf("Restarting service %q\n", e.sn)
	runner, err := e.serviceRunner()
	if err != nil {
		return fmt.Errorf("failed to get service runner: %w", err)
	}
	if err := runner.Restart(); err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}
	e.printf("Restarted service %q\n", e.sn)
	return nil
}

func (e *ttyExecer) editCmdFunc(c *cobra.Command, _ []string) error {
	st, err := e.s.serviceType(e.sn)
	if err != nil {
		return err
	}

	sv, err := e.s.serviceView(e.sn)
	if err != nil {
		return err
	}
	editEnv, _ := c.PersistentFlags().GetBool("env")
	editConfig, _ := c.PersistentFlags().GetBool("config")

	var srcPath string

	editConfigFn := func(cfg any) error {
		bs, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal systemd config: %w", err)
		}
		srcf, err := createTmpFile()
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer srcf.Close()
		srcPath = srcf.Name()
		if _, err := io.Copy(srcf, bytes.NewReader(bs)); err != nil {
			return fmt.Errorf("failed to write to temp file: %w", err)
		}
		return nil
	}

	var systemdUnitsBeingEdited []string
	af := sv.AsStruct().Artifacts
	if editEnv {
		srcPath, _ = af.Latest(db.ArtifactEnvFile)
	} else if editConfig {
		if err := editConfigFn(sv); err != nil {
			return fmt.Errorf("failed to edit config: %w", err)
		}
	} else {
		switch st {
		case db.ServiceTypeDockerCompose:
			srcPath, _ = af.Latest(db.ArtifactDockerComposeFile)
		case db.ServiceTypeSystemd:
			if len(af) == 0 {
				return fmt.Errorf("no unit files found")
			}
			srcf, err := createTmpFile()
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			defer srcf.Close()

			count := 0
			for _, name := range []db.ArtifactName{db.ArtifactSystemdUnit, db.ArtifactSystemdTimerFile} {
				path, ok := af.Latest(name)
				if !ok {
					continue
				}
				if count > 0 {
					fmt.Fprintf(srcf, "\n\n")
				}
				fmt.Fprintf(srcf, editUnitsSeparator, name)
				fmt.Fprintf(srcf, "\n\n")
				systemdUnitsBeingEdited = append(systemdUnitsBeingEdited, path)
				f, err := os.Open(path)
				if err != nil {
					return fmt.Errorf("failed to open unit file: %w", err)
				}
				if _, err := io.Copy(srcf, f); err != nil {
					return fmt.Errorf("failed to write to temp file: %w", err)
				}
				count++
			}
			if err := srcf.Close(); err != nil {
				return fmt.Errorf("failed to close temp file: %w", err)
			}
			srcPath = srcf.Name()
		}
	}

	tmpPath, err := copyToTmpFile(srcPath)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	if err := e.editFile(tmpPath); err != nil {
		return fmt.Errorf("failed to edit file: %w", err)
	}

	if same, err := fileutil.Identical(srcPath, tmpPath); err != nil {
		return err
	} else if same {
		e.printf("No changes detected\n")
		return nil
	}

	if editConfig {
		bs, err := os.ReadFile(tmpPath)
		if err != nil {
			return fmt.Errorf("failed to read temp file: %w", err)
		}
		var s2 db.Service
		if err := json.Unmarshal(bs, &s2); err != nil {
			return fmt.Errorf("failed to unmarshal temp file: %w", err)
		}
		_, _, err = e.s.cfg.DB.MutateService(e.sn, func(d *db.Data, s *db.Service) error {
			*s = s2
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to update service: %w", err)
		}
		i, err := e.s.NewInstaller(e.installerCfg())
		if err != nil {
			return fmt.Errorf("failed to create installer: %w", err)
		}
		i.NewCmd = e.newCmd
		return i.InstallGen(s2.Generation)
	}

	installFile := func() error {
		f, err := os.Open(tmpPath)
		if err != nil {
			return fmt.Errorf("failed to open temp file: %w", err)
		}
		defer f.Close()
		icfg := e.fileInstaller(c, nil)
		icfg.EnvFile = editEnv
		fi, err := NewFileInstaller(e.s, icfg)
		if err != nil {
			return fmt.Errorf("failed to create installer: %w", err)
		}
		defer fi.Close()
		if _, err := io.Copy(fi, f); err != nil {
			fi.Fail()
			return fmt.Errorf("failed to copy temp file to installer: %w", err)
		}
		return fi.Close()
	}

	switch st {
	case db.ServiceTypeDockerCompose:
		if editConfig {
			return fmt.Errorf("not implemented")
		}
		return installFile()
	case db.ServiceTypeSystemd:
		if editConfig {
			return fmt.Errorf("not implemented")
		}
		if editEnv {
			return installFile()
		}
		bs, err := os.ReadFile(tmpPath)
		if err != nil {
			return fmt.Errorf("failed to read temp file: %w", err)
		}
		submatches := editUnitsSeparatorRe.FindAllSubmatch(bs, -1)
		separateContents := editUnitsSeparatorRe.Split(string(bs), -1)
		if len(separateContents) < 1 {
			return fmt.Errorf("no unit files found")
		}
		separateContents = separateContents[1:] // Skip the first split which is empty
		if len(separateContents) != len(systemdUnitsBeingEdited) || len(submatches) != len(systemdUnitsBeingEdited) {
			return fmt.Errorf("mismatched number of unit files and contents")
		}
		newArtifacts := make(map[db.ArtifactName]string)
		for i, content := range separateContents {
			name := string(submatches[i][1])
			content = strings.TrimSpace(content)
			tmpf, err := createTmpFile()
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			defer os.Remove(tmpf.Name())
			defer tmpf.Close()
			if _, err := tmpf.WriteString(content); err != nil {
				return fmt.Errorf("failed to write to temp file: %w", err)
			}
			if err := tmpf.Close(); err != nil {
				return fmt.Errorf("failed to close temp file: %w", err)
			}
			p, ok := af.Latest(db.ArtifactName(name))
			if !ok {
				return fmt.Errorf("no unit file found for %q", name)
			}
			binPath := fileutil.UpdateVersion(p)
			if err := fileutil.CopyFile(tmpf.Name(), binPath); err != nil {
				return fmt.Errorf("failed to copy temp file to binary path: %w", err)
			}
			newArtifacts[db.ArtifactName(name)] = binPath
		}
		_, _, err = e.s.cfg.DB.MutateService(e.sn, func(d *db.Data, s *db.Service) error {
			for name, path := range newArtifacts {
				s.Artifacts[name].Refs["staged"] = path
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to update artifacts: %w", err)
		}
		i, err := e.s.NewInstaller(e.installerCfg())
		if err != nil {
			return fmt.Errorf("failed to create installer: %w", err)
		}
		i.NewCmd = e.newCmd
		return i.Install()
	default:
		return fmt.Errorf("unsupported service type: %v", st)
	}
}

func createTmpFile() (*os.File, error) {
	return os.CreateTemp("", "catch-tmp-*")
}

func copyToTmpFile(src string) (string, error) {
	tmpf, err := createTmpFile()
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	if src != "" {
		if err := fileutil.CopyFile(src, tmpf.Name()); err != nil {
			return "", fmt.Errorf("failed to copy file: %w", err)
		}
	}
	tmpf.Close()
	return tmpf.Name(), nil
}

func setWinsize(f *os.File, w, h int) {
	unix.IoctlSetWinsize(int(f.Fd()), syscall.TIOCSWINSZ, &unix.Winsize{
		Row: uint16(h),
		Col: uint16(w),
	})
}

func (e *ttyExecer) editFile(path string) error {
	if !e.isPty {
		return fmt.Errorf("edit requires a pty, please run ssh with -t")
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	cmd := e.newCmd(editor, path)
	cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", e.ptyReq.Term))
	return cmd.Run()
}

func (s *Server) envFile(sv db.ServiceView, staged bool) (string, error) {
	af := sv.AsStruct().Artifacts
	ef, _ := af.Latest(db.ArtifactEnvFile)
	return ef, nil
}

func (s *Server) printEnv(w io.Writer, sv db.ServiceView, staged bool) error {
	ef, err := s.envFile(sv, staged)
	if err != nil {
		return err
	}
	if ef == "" {
		return fmt.Errorf("no env file found")
	}
	b, err := os.ReadFile(ef)
	if err != nil {
		return fmt.Errorf("failed to read env file: %w", err)
	}
	fmt.Fprintf(w, "%s\n", b)
	return nil
}

func (e *ttyExecer) envCmdFunc(_ *cobra.Command, _ []string) error {
	sv, err := e.s.serviceView(e.sn)
	if err != nil {
		return err
	}
	return e.s.printEnv(e.rw, sv, false)
}

func (e *ttyExecer) enableCmdFunc(_ *cobra.Command, _ []string) error {
	if e.sn == SystemService || e.sn == CatchService {
		return fmt.Errorf("cannot install, reserved service name")
	}
	runner, err := e.serviceRunner()
	if err != nil {
		return err
	}
	enabler, ok := runner.(ServiceEnabler)
	if !ok {
		return fmt.Errorf("service does not support enable")
	}
	return enabler.Enable()
}

func (e *ttyExecer) disableCmdFunc(_ *cobra.Command, _ []string) error {
	if e.sn == SystemService || e.sn == CatchService {
		return fmt.Errorf("cannot disable system service")
	}

	runner, err := e.serviceRunner()
	if err != nil {
		return err
	}
	enabler, ok := runner.(ServiceEnabler)
	if !ok {
		return fmt.Errorf("service does not support disable")
	}
	return enabler.Disable()
}

func (e *ttyExecer) logsCmdFunc(cmd *cobra.Command, _ []string) error {
	// We don't support logs on the system service.
	if e.sn == SystemService {
		return fmt.Errorf("cannot show logs for system service")
	}
	// TODO(shayne): Make tailing optional
	runner, err := e.serviceRunner()
	if err != nil {
		return fmt.Errorf("failed to get service runner: %w", err)
	}
	follow, _ := cmd.Flags().GetBool("follow")
	lines, _ := cmd.Flags().GetInt("lines")
	return runner.Logs(&svc.LogOptions{Follow: follow, Lines: lines})
}

func (e *ttyExecer) statusCmdFunc(cmd *cobra.Command, _ []string) error {
	formatOut, _ := cmd.Flags().GetString("format")

	dv, err := e.s.cfg.DB.Get()
	if err != nil {
		return fmt.Errorf("failed to get services: %w", err)
	}
	if !dv.Valid() {
		return fmt.Errorf("no services found")
	}

	var statuses []ServiceStatusData

	if e.sn == SystemService {
		systemdStatuses, err := e.s.SystemdStatuses()
		if err != nil {
			return fmt.Errorf("failed to get systemd statuses: %w", err)
		}
		for sn, status := range systemdStatuses {
			service, err := e.s.serviceView(sn)
			if err != nil {
				return err
			}
			statuses = append(statuses, ServiceStatusData{
				ServiceName: sn,
				ServiceType: ServiceDataTypeFromServiceType(service.ServiceType()),
				ComponentStatus: []ComponentStatusData{
					{
						Name:   sn,
						Status: ComponentStatusFromServiceStatus(status),
					},
				},
			})
		}
		composeStatuses, err := e.s.DockerComposeStatuses()
		if err != nil {
			return fmt.Errorf("failed to get all docker compose statuses: %w", err)
		}
		for sn, cs := range composeStatuses {
			if len(cs) == 0 {
				statuses = append(statuses, ServiceStatusData{
					ServiceName: sn,
					ServiceType: ServiceDataTypeDocker,
					ComponentStatus: []ComponentStatusData{
						{
							Name:   sn,
							Status: ComponentStatusUnknown,
						},
					},
				})
				continue
			}
			data := ServiceStatusData{
				ServiceName:     sn,
				ServiceType:     ServiceDataTypeDocker,
				ComponentStatus: []ComponentStatusData{},
			}
			for cn, status := range cs {
				data.ComponentStatus = append(data.ComponentStatus, ComponentStatusData{
					Name:   cn,
					Status: ComponentStatusFromServiceStatus(status),
				})
			}
			statuses = append(statuses, data)
		}
	} else {
		st, err := e.s.serviceType(e.sn)
		if err != nil {
			return fmt.Errorf("failed to get service type: %w", err)
		}
		data := ServiceStatusData{
			ServiceName:     e.sn,
			ServiceType:     ServiceDataTypeFromServiceType(st),
			ComponentStatus: []ComponentStatusData{},
		}
		switch st {
		case db.ServiceTypeSystemd:
			status, err := e.s.SystemdStatus(e.sn)
			if err != nil {
				return fmt.Errorf("failed to get systemd status: %w", err)
			}
			data.ComponentStatus = append(data.ComponentStatus, ComponentStatusData{
				Name:   e.sn,
				Status: ComponentStatusFromServiceStatus(status),
			})
		case db.ServiceTypeDockerCompose:
			cs, err := e.s.DockerComposeStatus(e.sn)
			if err != nil {
				return fmt.Errorf("failed to get docker compose statuses: %w", err)
			}
			if len(cs) == 0 {
				data.ComponentStatus = append(data.ComponentStatus, ComponentStatusData{
					Name:   e.sn,
					Status: ComponentStatusUnknown,
				})
				return nil
			}
			for cn, status := range cs {
				data.ComponentStatus = append(data.ComponentStatus, ComponentStatusData{
					Name:   cn,
					Status: ComponentStatusFromServiceStatus(status),
				})
			}
		}
		statuses = append(statuses, data)
	}
	slices.SortFunc(statuses, func(a, b ServiceStatusData) int {
		return strings.Compare(a.ServiceName, b.ServiceName)
	})
	for _, status := range statuses {
		slices.SortFunc(status.ComponentStatus, func(a, b ComponentStatusData) int {
			return strings.Compare(a.Name, b.Name)
		})
	}

	if formatOut == "json" {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(statuses)
	}
	if formatOut == "json-pretty" {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(statuses)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "SERVICE\tTYPE\tCONTAINER\tSTATUS\t")

	for _, status := range statuses {
		for _, component := range status.ComponentStatus {
			if status.ServiceType == ServiceDataTypeDocker {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", status.ServiceName, status.ServiceType, component.Name, component.Status)
			} else {
				fmt.Fprintf(w, "%s\t%s\t-\t%s\t\n", status.ServiceName, status.ServiceType, component.Status)
			}
		}
	}
	return nil
}

func (e *ttyExecer) cronCmdFunc(cmd *cobra.Command, cronexpr string, args []string) error {
	oncal, err := cronutil.CronToCalender(cronexpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	cfg := e.fileInstaller(cmd, args)
	cfg.Timer = &svc.TimerConfig{
		OnCalendar: oncal,
		Persistent: true, // This should be an option keyvalue in the future
	}
	return e.install(cmd.InOrStdin(), cfg)
}

func (e *ttyExecer) removeCmdFunc(_ *cobra.Command, _ []string) error {
	if e.sn == SystemService || e.sn == CatchService {
		return fmt.Errorf("cannot remove system service")
	}
	runner, err := e.serviceRunner()
	if err != nil {
		if errors.Is(err, errNoServiceConfigured) {
			if err := e.s.RemoveService(e.sn); err != nil {
				return fmt.Errorf("failed to cleanup service %q: %w", e.sn, err)
			}
			e.printf("service %q not found\n", e.sn)
			return nil
		}
		return fmt.Errorf("failed to get service runner: %w", err)
	}
	// Confirm the removal of the service.
	if ok, err := cmdutil.Confirm(e.rw, e.rw, fmt.Sprintf("Are you sure you want to remove service %q?", e.sn)); err != nil {
		return fmt.Errorf("failed to confirm removal: %w", err)
	} else if !ok {
		return nil
	}

	err = runner.Remove()
	if err != nil && errors.Is(err, svc.ErrNotInstalled) {
		// Systemd service is not installed
		e.printf("warning: systemd service %q was not installed\n", e.sn)
	} else if err != nil {
		return fmt.Errorf("failed to remove service: %w", err)
	}
	err = e.s.RemoveService(e.sn)
	if err != nil {
		return fmt.Errorf("failed to cleanup service %q: %w", e.sn, err)
	}
	return nil
}

// ServiceRunner is an interface for the minimal set of methods required to
// manage a service.
type ServiceRunner interface {
	SetNewCmd(func(string, ...string) *exec.Cmd)

	Start() error
	Stop() error
	Restart() error

	Logs(opts *svc.LogOptions) error

	Remove() error
}

// ServiceEnabler is an interface extension for services that can be enabled and
// disabled.
type ServiceEnabler interface {
	Enable() error
	Disable() error
}

func (e *ttyExecer) newCmd(name string, args ...string) *exec.Cmd {
	c := exec.CommandContext(e.ctx, name, args...)
	rw := e.rw

	c.Stdin = rw
	c.Stdout = rw
	c.Stderr = rw

	if e.isPty {
		c.Env = append(c.Env, fmt.Sprintf("TERM=%s", e.ptyReq.Term))
		c.SysProcAttr = &syscall.SysProcAttr{
			Setctty: true,
			Setsid:  true,
		}
	}
	return c
}

func (e *ttyExecer) serviceRunner() (ServiceRunner, error) {
	st, err := e.s.serviceType(e.sn)
	if err != nil {
		return nil, fmt.Errorf("failed to get service type: %w", err)
	}
	var service ServiceRunner
	switch st {
	case db.ServiceTypeSystemd:
		systemd, err := e.s.systemdService(e.sn)
		if err != nil {
			return nil, err
		}
		service = &systemdServiceRunner{SystemdService: systemd}
	case db.ServiceTypeDockerCompose:
		docker, err := e.s.dockerComposeService(e.sn)
		if err != nil {
			return nil, err
		}
		service = &dockerComposeServiceRunner{DockerComposeService: docker}
	default:
		return nil, fmt.Errorf("unhandled service type %q", st)
	}
	if service != nil {
		service.SetNewCmd(e.newCmd)
	}
	return service, nil
}

type systemdServiceRunner struct {
	*svc.SystemdService
	newCmd func(string, ...string) *exec.Cmd
}

func (s *systemdServiceRunner) SetNewCmd(f func(string, ...string) *exec.Cmd) {
	s.newCmd = f
}

func (s *systemdServiceRunner) Start() error {
	return s.SystemdService.Start()
}

func (s *systemdServiceRunner) Stop() error {
	return s.SystemdService.Stop()
}

func (s *systemdServiceRunner) Restart() error {
	return s.SystemdService.Restart()
}

// Enable enables the service and starts it.
func (s *systemdServiceRunner) Enable() error {
	if err := s.SystemdService.Enable(); err != nil {
		return err
	}
	return s.SystemdService.Start()
}

// Disable stops and disables the service.
func (s *systemdServiceRunner) Disable() error {
	if err := s.SystemdService.Stop(); err != nil {
		return err
	}
	return s.SystemdService.Disable()
}

func (s *systemdServiceRunner) Logs(opts *svc.LogOptions) error {
	if opts == nil {
		opts = &svc.LogOptions{}
	}
	args := []string{"--no-pager", "--output=cat"}
	if opts.Follow {
		args = append(args, "--follow")
	}
	if opts.Lines > 0 {
		args = append(args, "--lines="+strconv.Itoa(opts.Lines))
	}
	args = append(args, "--unit="+s.SystemdService.Name())
	c := s.newCmd("journalctl", args...)
	if err := c.Start(); err != nil {
		return fmt.Errorf("failed to start journalctl: %w", err)
	}
	if err := c.Wait(); err != nil {
		return fmt.Errorf("failed to wait for journalctl: %w", err)
	}
	return nil
}

func (s *systemdServiceRunner) Remove() error {
	if err := s.SystemdService.Stop(); err != nil {
		return err
	}
	return s.SystemdService.Uninstall()
}

type dockerComposeServiceRunner struct {
	*svc.DockerComposeService
}

func (s *dockerComposeServiceRunner) SetNewCmd(f func(string, ...string) *exec.Cmd) {
	s.NewCmd = f
}

func (s *dockerComposeServiceRunner) Start() error {
	return s.DockerComposeService.Start()
}

func (s *dockerComposeServiceRunner) Stop() error {
	return s.DockerComposeService.Stop()
}

func (s *dockerComposeServiceRunner) Restart() error {
	return s.DockerComposeService.Restart()
}

func (s *dockerComposeServiceRunner) Logs(opts *svc.LogOptions) error {
	return s.DockerComposeService.Logs(opts)
}

func (s *dockerComposeServiceRunner) Remove() error {
	return s.DockerComposeService.Remove()
}

// Add this method to the ttyExecer struct
func (e *ttyExecer) eventsCmdFunc(cmd *cobra.Command, _ []string) error {
	ch := make(chan Event)
	all, _ := cmd.Flags().GetBool("all")
	defer e.s.RemoveEventListener(e.s.AddEventListener(ch, func(et Event) bool {
		if all {
			return true
		}
		return et.ServiceName == e.sn
	}))

	for {
		select {
		case event := <-ch:
			e.printf("Received event: %v\n", event)
		case <-e.ctx.Done():
			return nil
		case <-cmd.Context().Done():
			return nil
		}
	}
}

func (e *ttyExecer) umountCmdFunc(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("invalid number of arguments")
	}
	mountName := args[0]
	dv, err := e.s.cfg.DB.Get()
	if err != nil {
		return fmt.Errorf("failed to get services: %w", err)
	}
	vol, ok := dv.Volumes().GetOk(mountName)
	if !ok {
		return fmt.Errorf("volume %q not found", mountName)
	}
	m := &systemdMounter{e: e, v: *vol.AsStruct()}
	if err := m.umount(); err != nil {
		return fmt.Errorf("failed to umount %s: %w", vol.Path(), err)
	}

	d := dv.AsStruct()
	delete(d.Volumes, mountName)
	if err := e.s.cfg.DB.Set(d); err != nil {
		return fmt.Errorf("failed to save data: %w", err)
	}

	return nil
}

func (e *ttyExecer) mountCmdFunc(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		dv, err := e.s.cfg.DB.Get()
		if err != nil {
			return fmt.Errorf("failed to get services: %w", err)
		}
		tw := tabwriter.NewWriter(e.rw, 0, 0, 3, ' ', 0)
		defer tw.Flush()
		fmt.Fprintln(tw, "NAME\tSRC\tPATH\tTYPE\tOPTS")
		for _, v := range dv.AsStruct().Volumes {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", v.Name, v.Src, v.Path, v.Type, v.Opts)
		}
		return nil
	}
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("invalid number of arguments")
	}
	source := args[0]
	_, srcPath, ok := strings.Cut(source, ":")
	if !ok {
		return fmt.Errorf("source %q must be in the format host:path", source)
	}
	var mountName string
	if len(args) == 1 {
		mountName = filepath.Base(srcPath)
	} else {
		mountName = args[1]
	}

	if strings.Contains(mountName, "/") {
		return fmt.Errorf("target cannot contain a /")
	}

	mountType, _ := cmd.Flags().GetString("type")
	// Check the appropriate mounter is installed by stating /sbin/mount.<type>.
	mountCmd := fmt.Sprintf("/sbin/mount.%s", mountType)
	if _, err := os.Stat(mountCmd); err != nil {
		return fmt.Errorf("mount command %q not found", mountCmd)
	}

	opts, _ := cmd.Flags().GetString("opts")
	target := filepath.Join(e.s.cfg.MountsRoot, mountName)
	dv, err := e.s.cfg.DB.Get()
	if err != nil {
		return fmt.Errorf("failed to get services: %w", err)
	}
	if dv.Volumes().Contains(mountName) {
		return fmt.Errorf("volume %q already exists; please remove it first", mountName)
	}
	deps, _ := cmd.Flags().GetStringSlice("deps")
	d := dv.AsStruct()
	vol := db.Volume{
		Name: mountName,
		Src:  source,
		Path: target,
		Type: mountType,
		Opts: opts,
		Deps: strings.Join(deps, " "),
	}
	mak.Set(&d.Volumes, mountName, &vol)
	if err := e.s.cfg.DB.Set(d); err != nil {
		return fmt.Errorf("failed to save data: %w", err)
	}
	m := &systemdMounter{v: vol}

	if err := m.mount(); err != nil {
		return fmt.Errorf("failed to mount %s at %s: %w", source, target, err)
	}

	fmt.Fprintf(e.rw, "Mounted %s at %s\n", source, target)
	return nil
}

var ipv4Re = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}/\d{1,2}\b`)

func parseIPv4Addresses(text string) []string {
	matches := ipv4Re.FindAllString(text, -1)

	var ips []string
	for _, match := range matches {
		ip := strings.Split(match, "/")[0]
		ips = append(ips, ip)
	}
	return ips
}

func (e *ttyExecer) tsCmdFunc(_ *cobra.Command, args []string) error {
	if e.sn == SystemService || e.sn == CatchService {
		return errors.New("ts command not supported for sys or catch service")
	}
	sv, err := e.s.serviceView(e.sn)
	if err != nil {
		return fmt.Errorf("failed to get service view: %w", err)
	}
	if !sv.TSNet().Valid() {
		return errors.New("service is not connected to tailscale")
	}
	sock := filepath.Join(e.s.serviceRunDir(e.sn), "tailscaled.sock")
	if _, err := os.Stat(sock); err != nil {
		return fmt.Errorf("tailscaled socket not found: %w", err)
	}
	ts, err := e.s.getTailscaleBinary(sv.TSNet().Version())
	if err != nil {
		return fmt.Errorf("failed to get tailscale binary: %w", err)
	}
	args = append([]string{
		"--socket=" + sock,
	}, args...)
	c := e.newCmd(ts, args...)
	if err := c.Run(); err != nil {
		return fmt.Errorf("failed to run tailscale command: %w", err)
	}
	return nil
}

func (e *ttyExecer) ipCmdFunc(_ *cobra.Command, _ []string) error {
	if e.sn == CatchService {
		st, err := e.s.cfg.LocalClient.StatusWithoutPeers(e.ctx)
		if err != nil {
			return fmt.Errorf("failed to get IP address: %w", err)
		}
		for _, ip := range st.TailscaleIPs {
			fmt.Fprintln(e.rw, ip)
		}
		return nil
	}

	args := []string{"-o", "-4", "addr", "list"}
	if e.sn != SystemService {
		sv, err := e.s.serviceView(e.sn)
		if err != nil {
			return fmt.Errorf("failed to get service view: %w", err)
		}
		if _, ok := sv.AsStruct().Artifacts.Gen(db.ArtifactNetNSService, sv.Generation()); ok {
			netns := fmt.Sprintf("yeet-%s-ns", e.sn)
			args = append([]string{"netns", "exec", netns, "ip"}, args...)
		}
	}
	c := exec.Command("ip", args...)
	bs, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get IP addresses: %w", err)
	}
	ips := parseIPv4Addresses(string(bs))
	for _, ip := range ips {
		// Skip 127.0.0.1
		if ip == "127.0.0.1" {
			continue
		}
		fmt.Fprintln(e.rw, ip)
	}
	return nil
}
