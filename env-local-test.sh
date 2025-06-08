#! /usr/bin/env bash
# Copyright 2023-2025 The Parca Authors
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

set -euo pipefail

ARCH_UNAME=$(uname -m)
if [ "$ARCH_UNAME" = "x86_64" ]; then
    ARCH=amd64
else
    ARCH=arm64
fi

curl -LO "https://storage.googleapis.com/minikube/releases/latest/minikube-linux-$ARCH"
chmod +x "./minikube-linux-$ARCH"
sudo mv "minikube-linux-$ARCH" /usr/local/bin/minikube

curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash

# renovate: datasource=github-releases depName=tilt-dev/ctlptl
CTLPTL_VERSION="v0.8.42"
curl -fsSL "https://github.com/tilt-dev/ctlptl/releases/download/${CTLPTL_VERSION}/ctlptl.${CTLPTL_VERSION#v}.linux.$(if [ $ARCH = amd64 ]; then echo "$ARCH_UNAME"; else echo "$ARCH"; fi).tar.gz" | sudo tar -xzv -C /usr/local/bin ctlptl
