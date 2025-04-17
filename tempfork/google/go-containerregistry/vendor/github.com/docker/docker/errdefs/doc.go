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

// Package errdefs defines a set of error interfaces that packages should use for communicating classes of errors.
// Errors that cross the package boundary should implement one (and only one) of these interfaces.
//
// Packages should not reference these interfaces directly, only implement them.
// To check if a particular error implements one of these interfaces, there are helper
// functions provided (e.g. `Is<SomeError>`) which can be used rather than asserting the interfaces directly.
// If you must assert on these interfaces, be sure to check the causal chain (`err.Cause()`).
package errdefs // import "github.com/docker/docker/errdefs"
