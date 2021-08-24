#! /usr/bin/env bash
set -euo pipefail

curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
chmod +x ./minikube-linux-amd64
sudo minikube-linux-amd64 /usr/local/bin/minikube

curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash

CTLPTL_VERSION="0.5.0"
curl -fsSL https://github.com/tilt-dev/ctlptl/releases/download/v$CTLPTL_VERSION/ctlptl.$CTLPTL_VERSION.linux.x86_64.tar.gz | sudo tar -xzv -C /usr/local/bin ctlptl
