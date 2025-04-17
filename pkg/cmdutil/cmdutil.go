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

package cmdutil

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func NewStdCmd(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func Confirm(r io.Reader, w io.Writer, msg string) (bool, error) {
	fmt.Fprintf(w, "%s [y/N]: ", msg)

	var confirm string
	_, err := fmt.Fscanln(r, &confirm)
	if err != nil && err.Error() != "unexpected newline" {
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}
	if strings.ToLower(confirm) != "y" {
		return false, nil
	}
	return true, nil
}
