#!/usr/bin/env bash

# Copyright 2024 The Parca Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# this is done to prevent the yarn build command from removing the keep.go file
# Create a temporary directory to store the keep.go file
mkdir -p tmp/ui-keep

# Copy the keep.go file to the temporary directory
cp packages/app/web/build/keep.go tmp/ui-keep/keep.go

# Run the yarn build command
command yarn run build

# Copy the keep.go file back to its original location
cp tmp/ui-keep/keep.go ui/packages/app/web/build/keep.go

# Remove the temporary directory
rm -rf tmp/ui-keep

exit 0
