# https://github.com/hadolint/hadolint/issues/861
# hadolint ignore=DL3029
FROM --platform="${BUILDPLATFORM:-linux/amd64}" docker.io/busybox:1.36.0@sha256:9e2bbca079387d7965c3a9cee6d0c53f4f4e63ff7637877a83c4c05f2a666112 as builder
RUN mkdir /.cache && touch -t 202101010000.00 /.cache

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG TARGETVARIANT=v1

# renovate: datasource=github-releases depName=grpc-ecosystem/grpc-health-probe
ARG GRPC_HEALTH_PROBE_VERSION=v0.4.18
# Downloading grpc_health_probe from github releases with retry as we have seen it fail a lot on ci.
RUN for i in `seq 1 50`; do \
    wget -qO/bin/grpc_health_probe "https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-${TARGETOS}-${TARGETARCH}" && \
    chmod +x /bin/grpc_health_probe && \
    break; \
    echo "Failed to download grpc_health_probe on $i th attempt, retrying in 5s..." \
    sleep 5; \
    done

WORKDIR /app
COPY dist dist

# NOTICE: See goreleaser.yml for the build paths.
RUN if [ "${TARGETARCH}" = 'amd64' ]; then \
        cp "dist/parca_${TARGETOS}_${TARGETARCH}_${TARGETVARIANT:-v1}/parca" . ; \
    elif [ "${TARGETARCH}" = 'arm' ]; then \
        cp "dist/parca_${TARGETOS}_${TARGETARCH}_${TARGETVARIANT##v}/parca" . ; \
    else \
        cp "dist/parca_${TARGETOS}_${TARGETARCH}/parca" . ; \
    fi
RUN chmod +x parca

# https://github.com/hadolint/hadolint/issues/861
# hadolint ignore=DL3029
FROM --platform="${TARGETPLATFORM:-linux/amd64}"  docker.io/alpine:3.18.0@sha256:02bb6f428431fbc2809c5d1b41eab5a68350194fb508869a33cb1af4444c9b11 AS runner

LABEL \
  org.opencontainers.image.source="https://github.com/parca-dev/parca" \
  org.opencontainers.image.url="https://github.com/parca-dev/parca" \
  org.opencontainers.image.description="Continuous profiling for analysis of CPU and memory usage, down to the line number and throughout time. Saving infrastructure cost, improving performance, and increasing reliability." \
  org.opencontainers.image.licenses="Apache-2.0"

USER nobody

COPY --chown=0:0 --from=builder /bin/grpc_health_probe /
COPY --chown=0:0 --from=builder /app/parca /parca
COPY --chown=0:0 parca.yaml /parca.yaml

CMD ["/parca"]
