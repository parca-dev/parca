#!/usr/bin/env bash
set +eox pipefail

MANIFEST="$1"
ARCHS=('amd64' 'arm64')

# SHA order is respectively ('amd64' 'arm64')
# this image is what docker.io/golang:1.18.2-alpine3.15 on May 12 2022
DOCKER_GOLANG_ALPINE_SHAS=(
    'docker.io/golang@sha256:3765360960a954c24b215518a41b0bf8e9e2fe3bd142b6f782fa56f8e00b8d21'
    'docker.io/golang@sha256:394c1e739574aac2b389c628d0a199316164c3ed9d821c06ebe4ad3f133ed1f9'
)

# SHA order is respectively ('amd64' 'arm64')
# this image is what docker.io/alpine:3.15.4 on May 11 2022.
# Here is how to obtain the digests:
# for r in amd64 arm64v8; do docker pull $r/alpine:3.15.4 | grep Digest; done
DOCKER_ALPINE_SHAS=(
    'docker.io/alpine@sha256:a777c9c66ba177ccfea23f2a216ff6721e78a662cd17019488c417135299cd89'
    'docker.io/alpine@sha256:f3bec467166fd0e38f83ff32fb82447f5e89b5abd13264a04454c75e11f1cdc6'
)

for i in "${!ARCHS[@]}"; do
    ARCH=${ARCHS[$i]}
    DOCKER_GOLANG_ALPINE_SHA=${DOCKER_GOLANG_ALPINE_SHAS[$i]}
    DOCKER_ALPINE_SHA=${DOCKER_ALPINE_SHAS[$i]}
    echo "Building manifest for $MANIFEST with arch \"$ARCH\""
    podman build \
        --build-arg GOLANG_BASE="$DOCKER_GOLANG_ALPINE_SHA" \
        --build-arg RUNNER_BASE="$DOCKER_ALPINE_SHA" \
        --arch "$ARCH" \
        --timestamp 0 \
        --manifest "$MANIFEST" .; \
done
