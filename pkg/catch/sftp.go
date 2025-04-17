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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
	"tailscale.com/syncs"
	gssh "tailscale.com/tempfork/gliderlabs/ssh"
)

// sftpHandler handles incoming SFTP subsystem requests.
type sftpHandler struct {
	server  *Server
	session gssh.Session
}

// newSFTPHandler creates a new sftpHandler instance. It accepts a new channel
// and returns a sftpHandler instance. NOTE: Other than the FilePut handler, all
// other handlers are sft.InMemHandlers.
func newSFTPHandler(server *Server, session gssh.Session) *sftpHandler {
	return &sftpHandler{
		server:  server,
		session: session,
	}
}

type fileHandler struct {
	s           *Server
	session     gssh.Session
	fileMapping syncs.Map[string, *FileInstaller]
}

func (f *fileHandler) Fileread(req *sftp.Request) (io.ReaderAt, error) {
	log.Printf("Fileread: %+v", req)
	if req.Method != "Get" {
		return nil, fmt.Errorf("unsupported method: %q", req.Method)
	}
	path, err := f.resolvePath(req.Filepath)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

// resolvePath validates the given path and returns the absolute path
// on the host filesystem.
func (f *fileHandler) resolvePath(fullPath string) (string, error) {
	sn, _, err := f.s.serviceAndUser(f.session)
	if err != nil {
		return "", err
	}
	if fullPath == "/env" || fullPath == "/stage/env" {
		sv, err := f.s.serviceView(sn)
		if err != nil {
			return "", err
		}
		ef, err := f.s.envFile(sv, fullPath == "/stage/env")
		if err != nil {
			return "", err
		}
		return ef, nil
	}
	path, ok := strings.CutPrefix(fullPath, "/data")
	if !ok {
		return "", fmt.Errorf("invalid path: %q", path)
	}
	if !strings.HasPrefix(path, "/") && path != "" {
		return "", fmt.Errorf("invalid path: %q", path)
	}
	if path == "/.env" {
		// The .env file is special and is stored in the service directory,
		// so reject any attempts to access it via /data.
		return "", fmt.Errorf("invalid path: %q", path)
	}

	svcDir := f.s.serviceRootDir(sn)
	return filepath.Join(svcDir, fullPath), nil
}

func (f *fileHandler) Filelist(req *sftp.Request) (_ sftp.ListerAt, err error) {
	log.Printf("Filelist: %+v", req)
	defer func() {
		log.Printf("Filelist: %v", err)
	}()
	path, err := f.resolvePath(req.Filepath)
	if err != nil {
		return nil, err
	}
	return &lister{
		method: req.Method,
		path:   path,
	}, nil
}

type lister struct {
	method string
	path   string
	de     []fs.DirEntry
}

func (ls *lister) ListAt(fis []os.FileInfo, off int64) (n int, err error) {
	defer func() {
		log.Printf("ListAt(%+v): %d, %v", ls, n, err)
	}()
	if ls.method == "Stat" {
		fi, err := os.Stat(ls.path)
		if err != nil {
			return 0, err
		}
		fis[0] = fi
		return 1, io.EOF
	}
	if ls.method != "List" {
		return 0, fmt.Errorf("unsupported method: %q", ls.method)
	}
	if ls.de == nil {
		de, err := os.ReadDir(ls.path)
		if err != nil {
			return 0, err
		}
		ls.de = de
	}
	count := 0
	for i, d := range ls.de[off:] {
		if i >= len(fis) {
			return count, nil
		}
		fi, err := d.Info()
		if err != nil {
			return count, err
		}
		fis[i] = fi
		count++
	}
	return count, io.EOF
}

func (f *fileHandler) Filecmd(req *sftp.Request) (ret error) {
	log.Printf("Filecmd: %+v", req)
	defer func() {
		log.Printf("Filecmd: %v", ret)
	}()
	if req.Method != "Setstat" {
		return fmt.Errorf("unsupported method: %q", req.Method)
	}
	log.Println("Setstat: ", req.Attributes())
	log.Println("AttrFlags:", req.AttrFlags())
	if _, ok := f.fileMapping.Load(req.Filepath); ok {
		return nil
	}
	if _, err := f.resolvePath(req.Filepath); err != nil {
		return err
	}

	return nil
}

// serve handles SFTP requests and delegates them to the pre-configured handlers.
func (s *sftpHandler) serve() error {
	log.Printf("SFTP session started: %s", s.session.User())
	fh := &fileHandler{
		s:       s.server,
		session: s.session,
	}
	server := sftp.NewRequestServer(noCloseSession{s.session}, sftp.Handlers{
		FilePut:  fh,
		FileGet:  fh,
		FileCmd:  fh,
		FileList: fh,
	})
	defer server.Close()
	if err := server.Serve(); err != nil && err != io.EOF {
		return fmt.Errorf("SFTP server completed with error: %w", err)
	}
	return nil
}

func (f *fileHandler) Filewrite(req *sftp.Request) (_ io.WriterAt, err error) {
	defer func() {
		if err != nil {
			log.Printf("Failed to handle SFTP request: %v", err)
		}
	}()
	log.Printf("User: %s", f.session.User())
	log.Printf("Received file: %s", req.Filepath)
	log.Printf("Target: %s", req.Target)
	log.Printf("Method: %s", req.Method)
	log.Printf("Flags: %d", req.Flags)
	log.Printf("Attrs: %s", req.Attrs)
	if req.Method != "Put" {
		return nil, fmt.Errorf("unsupported method: %q", req.Method)
	}
	if strings.HasPrefix(req.Filepath, "/data/") {
		return f.uploadFile(req.Filepath)
	}
	var fs *FileInstaller
	switch req.Filepath {
	case "/", "/stage":
		fs, err = f.binFile(req.Filepath == "/")
	case "/env", "/stage/env":
		fs, err = f.envFile(req.Filepath == "/env")
	default:
		return nil, fmt.Errorf("unsupported path: %q", req.Filepath)
	}
	if err != nil {
		return nil, err
	}
	f.fileMapping.Store(req.Filepath, fs)
	return fs, nil
}

func (f *fileHandler) uploadFile(dst string) (io.WriterAt, error) {
	sn, user, err := f.s.serviceAndUser(f.session)
	if err != nil {
		return nil, err
	}
	// Ensure the directories exist, in case we haven't seen this service
	// before.
	if err := f.s.ensureDirs(sn, user); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	path, err := f.resolvePath(dst)
	if err != nil {
		return nil, err
	}
	pf, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return nil, err
	}
	return pf, nil
}

func (f *fileHandler) envFile(install bool) (*FileInstaller, error) {
	sn, user, err := f.s.serviceAndUser(f.session)
	if err != nil {
		return nil, err
	}
	if _, err := f.s.serviceView(sn); err != nil {
		if !errors.Is(err, errServiceNotFound) {
			return nil, err
		}
		install = false // only stage env file if service does not exist
	}
	return NewFileInstaller(f.s, FileInstallerCfg{
		EnvFile: true,
		InstallerCfg: InstallerCfg{
			ServiceName:      sn,
			SSHSessionCloser: f.session,
			User:             user,
		},
		StageOnly: !install,
	})
}

func (f *fileHandler) binFile(install bool) (*FileInstaller, error) {
	sn, user, err := f.s.serviceAndUser(f.session)
	if err != nil {
		return nil, err
	}
	return NewFileInstaller(f.s, FileInstallerCfg{
		InstallerCfg: InstallerCfg{
			ServiceName:      sn,
			SSHSessionCloser: f.session,
			User:             user,
		},
		StageOnly: !install,
	})
}
