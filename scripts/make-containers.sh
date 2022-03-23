#!/usr/bin/env bash
set +x

VERSION="$1"
COMMIT="$2"
MANIFEST="$3"
ARCHS=('amd64' 'arm64' 'armv6' 'armv7' '386')

# SHA order is respectively ('amd64' 'arm64' 'armv6' 'armv7' '386')
# this image is what docker.io/golang:1.18.0-alpine3.15 on April 7 2022
DOCKER_GOLANG_ALPINE_SHAS=(
    'docker.io/golang@sha256:7473adb02bd430045c938f61e2c2177ff62b28968579dfed99085a0960f76f5d'
    'docker.io/golang@sha256:fe66e641602dfb60f5c746b57bde60fcafce94b44f2bc2ca2bdc1e3711b24911'
    'docker.io/golang@sha256:674352693f4ce096d73fa32d81642ff004c3363caa43fbda13f30e60c564fd9d'
    'docker.io/golang@sha256:1661ccca6a506959ff7ceee79ac0278785c71403de8cb03b5ad1672f466ab769'
    'docker.io/golang@sha256:f65787ec108a90d8af2017a8685ae94f70205b24122fc5863f117d78c235e72f'
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
        --manifest "$MANIFEST" .; \
done
