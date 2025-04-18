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

package sockets

import (
	"net"
	"net/http"
	"time"

	"github.com/Microsoft/go-winio"
)

func configureUnixTransport(tr *http.Transport, proto, addr string) error {
	return ErrProtocolNotAvailable
}

func configureNpipeTransport(tr *http.Transport, proto, addr string) error {
	// No need for compression in local communications.
	tr.DisableCompression = true
	tr.Dial = func(_, _ string) (net.Conn, error) {
		return DialPipe(addr, defaultTimeout)
	}
	return nil
}

// DialPipe connects to a Windows named pipe.
func DialPipe(addr string, timeout time.Duration) (net.Conn, error) {
	return winio.DialPipe(addr, &timeout)
}
