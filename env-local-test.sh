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

# renovate: datasource=github-releases depName=tilt-dev/ctlptl
CTLPTL_VERSION="v0.8.7"
curl -fsSL "https://github.com/tilt-dev/ctlptl/releases/download/${CTLPTL_VERSION}/ctlptl.${CTLPTL_VERSION#v}.linux.$(if [ $ARCH = amd64 ]; then echo "$ARCH_UNAME"; else echo "$ARCH"; fi).tar.gz" | sudo tar -xzv -C /usr/local/bin ctlptl
