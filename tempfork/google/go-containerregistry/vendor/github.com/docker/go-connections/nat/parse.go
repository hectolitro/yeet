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

package nat

import (
	"fmt"
	"strconv"
	"strings"
)

// PartParser parses and validates the specified string (data) using the specified template
// e.g. ip:public:private -> 192.168.0.1:80:8000
// DEPRECATED: do not use, this function may be removed in a future version
func PartParser(template, data string) (map[string]string, error) {
	// ip:public:private
	var (
		templateParts = strings.Split(template, ":")
		parts         = strings.Split(data, ":")
		out           = make(map[string]string, len(templateParts))
	)
	if len(parts) != len(templateParts) {
		return nil, fmt.Errorf("Invalid format to parse. %s should match template %s", data, template)
	}

	for i, t := range templateParts {
		value := ""
		if len(parts) > i {
			value = parts[i]
		}
		out[t] = value
	}
	return out, nil
}

// ParsePortRange parses and validates the specified string as a port-range (8000-9000)
func ParsePortRange(ports string) (uint64, uint64, error) {
	if ports == "" {
		return 0, 0, fmt.Errorf("Empty string specified for ports.")
	}
	if !strings.Contains(ports, "-") {
		start, err := strconv.ParseUint(ports, 10, 16)
		end := start
		return start, end, err
	}

	parts := strings.Split(ports, "-")
	start, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return 0, 0, err
	}
	end, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return 0, 0, err
	}
	if end < start {
		return 0, 0, fmt.Errorf("Invalid range specified for the Port: %s", ports)
	}
	return start, end, nil
}
