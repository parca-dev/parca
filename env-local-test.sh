#! /usr/bin/env bash
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

CTLPTL_VERSION="0.5.1"
curl -fsSL "https://github.com/tilt-dev/ctlptl/releases/download/v$CTLPTL_VERSION/ctlptl.$CTLPTL_VERSION.linux.$ARCH.tar.gz" | sudo tar -xzv -C /usr/local/bin ctlptl
