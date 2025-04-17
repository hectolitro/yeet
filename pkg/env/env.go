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

package env

import (
	"fmt"
	"io"
	"os"
	"reflect"
)

// Write writes an environment file with the given name and content.
func Write(name string, e any) error {
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer f.Close()
	if err := marshalEnv(f, e); err != nil {
		return fmt.Errorf("failed to marshal env: %v", err)
	}
	return f.Close()
}

func marshalEnv(o io.Writer, e any) error {
	re := reflect.ValueOf(e)
	if re.Kind() == reflect.Ptr {
		re = re.Elem()
	}
	ret := re.Type()
	for i := 0; i < re.NumField(); i++ {
		field := re.Field(i)
		tag := ret.Field(i).Tag.Get("env")
		if tag == "" {
			continue
		}
		if field.IsZero() {
			continue
		}
		fmt.Fprintf(o, "%s=%s\n", tag, field.Interface())
	}
	return nil
}
