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

package api // import "github.com/docker/docker/api"

// MinVersion represents Minimum REST API version supported
// Technically the first daemon API version released on Windows is v1.25 in
// engine version 1.13. However, some clients are explicitly using downlevel
// APIs (e.g. docker-compose v2.1 file format) and that is just too restrictive.
// Hence also allowing 1.24 on Windows.
const MinVersion string = "1.24"
