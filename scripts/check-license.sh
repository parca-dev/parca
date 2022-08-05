#!/usr/bin/env bash
#
# Copyright 2022 The Parca Authors
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

# exit immediately when a command fails
set -e
# only exit with zero if all commands of the pipeline exit successfully
set -o pipefail
# error on unset variables
set -u

licRes=$(
    find . -type f -iname '*.go' ! -path '*/vendor/*' ! -path '*/tmp*' ! -path '*/internal*' -exec sh -c 'head -n3 $1 | grep -Eq "(Copyright|generated|GENERATED)" || echo -e  $1' {} {} \;
)

if [ -n "${licRes}" ]; then
    echo -e "license header checking failed:\\n${licRes}"
    exit 255
fi
