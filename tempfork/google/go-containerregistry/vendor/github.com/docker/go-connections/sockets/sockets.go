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

// Package sockets provides helper functions to create and configure Unix or TCP sockets.
package sockets

import (
	"errors"
	"net"
	"net/http"
	"time"
)

// Why 32? See https://github.com/docker/docker/pull/8035.
const defaultTimeout = 32 * time.Second

// ErrProtocolNotAvailable is returned when a given transport protocol is not provided by the operating system.
var ErrProtocolNotAvailable = errors.New("protocol not available")

// ConfigureTransport configures the specified Transport according to the
// specified proto and addr.
// If the proto is unix (using a unix socket to communicate) or npipe the
// compression is disabled.
func ConfigureTransport(tr *http.Transport, proto, addr string) error {
	switch proto {
	case "unix":
		return configureUnixTransport(tr, proto, addr)
	case "npipe":
		return configureNpipeTransport(tr, proto, addr)
	default:
		tr.Proxy = http.ProxyFromEnvironment
		dialer, err := DialerFromEnvironment(&net.Dialer{
			Timeout: defaultTimeout,
		})
		if err != nil {
			return err
		}
		tr.Dial = dialer.Dial
	}
	return nil
}
