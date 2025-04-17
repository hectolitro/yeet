#!/bin/bash
# Copyright 2025 AUTHORS
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

if [[ $GO111MODULE == "on" ]]; then
  go get .
else
  go get -u -v $(go list -f '{{join .Imports "\n"}}{{"\n"}}{{join .TestImports "\n"}}' ./... | sort | uniq | grep -v appengine)
fi

if [[ $GOAPP == "true" ]]; then
  mkdir /tmp/sdk
  curl -o /tmp/sdk.zip "https://storage.googleapis.com/appengine-sdks/featured/go_appengine_sdk_linux_amd64-1.9.68.zip"
  unzip -q /tmp/sdk.zip -d /tmp/sdk
  # NOTE: Set the following env vars in the test script:
  # export PATH="$PATH:/tmp/sdk/go_appengine"
  # export APPENGINE_DEV_APPSERVER=/tmp/sdk/go_appengine/dev_appserver.py
fi

