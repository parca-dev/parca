#!/usr/bin/env bash
set +x

VERSION="$1"
COMMIT="$2"
MANIFEST="$3"
ARCHS=('amd64' 'arm64' 'armv6' 'armv7' '386')
NPROC=$(nproc --all)

# SHA order is respectively ('amd64' 'arm64' 'armv6' 'armv7' '386')
# this image is what docker.io/golang:1.17.8-alpine3.15 on March 14 2021
DOCKER_GOLANG_ALPINE_SHAS=(
    'docker.io/golang@sha256:e2e68a9cdd5da82458652fdac3908a3a270686b38039f2829855398e2e06019d'
    'docker.io/golang@sha256:a55e3394bce5523e660d9ee17adb827731ddd02559e75aabe3c5417778f8941e'
    'docker.io/golang@sha256:3eb758fdfa1fa203d5503921d5bc18765c733add0cfbf98a3302e6a055cebaa1'
    'docker.io/golang@sha256:96b13a554921ea4e4d4435244497708779e9047863c77fcc89892b4fd5ef4ac1'
    'docker.io/golang@sha256:46a83fb9832e1aeed048139e4bc63bbfe039462ea12114dc2d1b2cd7574db7b2'
)

# SHA order is respectively ('amd64' 'arm64' 'armv6' 'armv7' '386')
# this image is what node:17.7.1-alpine3.15 is on March 14 2021
DOCKER_NODE_ALPINE_SHAS=(
    'docker.io/library/node@sha256:10ef59da5b5ccdbaff99a81df1bcccb0500723633ce406efed6f1fb74adc8568'
    'docker.io/library/node@sha256:8ca33e44fa0be3989b4dbced274f0fa17cc9ded52e3c0824725264b32bdf7c38'
    'docker.io/library/node@sha256:9d674d6222f95556b5a4e2162cc09450928dec5979b99f208b6b0526c6c41e98'
    'docker.io/library/node@sha256:6c8a221b76847a08a3789c2a18fa194e6a5825a7a1037f3d47396fbdf7cfeca7'
    'docker.io/library/node@sha256:10ef59da5b5ccdbaff99a81df1bcccb0500723633ce406efed6f1fb74adc8568' # this is an amd64 image, unfortunately there is no 386 image for nodejs.
)

# SHA order is respectively ('amd64' 'arm64' 'armv6' 'armv7' '386')
# this image is what docker.io/alpine:3.15.0 on March 14 2021
DOCKER_ALPINE_SHAS=(
    'docker.io/alpine@sha256:e7d88de73db3d3fd9b2d63aa7f447a10fd0220b7cbf39803c803f2af9ba256b3'
    'docker.io/alpine@sha256:c74f1b1166784193ea6c8f9440263b9be6cae07dfe35e32a5df7a31358ac2060'
    'docker.io/alpine@sha256:e047bc2af17934d38c5a7fa9f46d443f1de3a7675546402592ef805cfa929f9d'
    'docker.io/alpine@sha256:8483ecd016885d8dba70426fda133c30466f661bb041490d525658f1aac73822'
    'docker.io/alpine@sha256:2689e157117d2da668ad4699549e55eba1ceb79cb7862368b30919f0488213f4'
)

for i in "${!ARCHS[@]}"; do
    ARCH=${ARCHS[$i]}
    DOCKER_GOLANG_ALPINE_SHA=${DOCKER_GOLANG_ALPINE_SHAS[$i]}
    DOCKER_NODE_ALPINE_SHA=${DOCKER_NODE_ALPINE_SHAS[$i]}
    DOCKER_ALPINE_SHA=${DOCKER_ALPINE_SHAS[$i]}
    echo "Building manifest for $MANIFEST with arch \"$ARCH\""
    podman build \
        --build-arg VERSION="$VERSION" \
        --build-arg COMMIT="$COMMIT" \
        --build-arg GOLANG_BUILDER_BASE="$DOCKER_GOLANG_ALPINE_SHA" \
        --build-arg NODE_BUILDER_BASE="$DOCKER_NODE_ALPINE_SHA" \
        --build-arg RUNNER_BASE="$DOCKER_ALPINE_SHA" \
        --build-arg ARCH="$ARCH" \
        --arch "$ARCH" \
        --timestamp 0 \
        --jobs "$NPROC" \
        --manifest "$MANIFEST" .; \
done
