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

package blkiodev // import "github.com/docker/docker/api/types/blkiodev"

import "fmt"

// WeightDevice is a structure that holds device:weight pair
type WeightDevice struct {
	Path   string
	Weight uint16
}

func (w *WeightDevice) String() string {
	return fmt.Sprintf("%s:%d", w.Path, w.Weight)
}

// ThrottleDevice is a structure that holds device:rate_per_second pair
type ThrottleDevice struct {
	Path string
	Rate uint64
}

func (t *ThrottleDevice) String() string {
	return fmt.Sprintf("%s:%d", t.Path, t.Rate)
}
