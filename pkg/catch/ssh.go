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
	"fmt"
	"io"
	"log"

	gssh "tailscale.com/tempfork/gliderlabs/ssh"
)

func (s *Server) SSHHandler(session gssh.Session) {
	rc := session.RawCommand()
	log.Printf("Received command: %s", rc)

	sn, user, err := s.serviceAndUser(session)
	if err != nil {
		fmt.Fprintf(session, "Error: %v\n", err)
		return
	}

	rwc := io.ReadWriteCloser(session)
	ptyReq, ptyWCh, isPty := session.Pty()
	execer := &ttyExecer{
		ctx:       session.Context(),
		s:         s,
		args:      session.Command(),
		sn:        sn,
		user:      user,
		rawRW:     rwc,
		isPty:     isPty,
		ptyReq:    ptyReq,
		ptyWCh:    ptyWCh,
		rawCloser: session,
	}

	if err := execer.run(); err != nil {
		session.Exit(1)
		return
	}

	session.Exit(0)
}
