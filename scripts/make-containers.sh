#!/usr/bin/env bash
set -euo pipefail

MANIFEST="${1?Image name must be provided}"
PLATFORMS=('linux/amd64' 'linux/arm64')

for platform in "${PLATFORMS[@]}"; do
    printf 'Building manifest for %s with platform "%s"\n' "${MANIFEST}" "${platform}"
    podman build \
        --platform "${platform}" \
        --timestamp 0 \
        --manifest "${MANIFEST}" .
done
