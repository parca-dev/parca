# https://github.com/hadolint/hadolint/issues/861
# hadolint ignore=DL3029
FROM --platform="${BUILDPLATFORM:-linux/amd64}" docker.io/busybox:1.35.0@sha256:ec6dc15985db1ffc5dc2b341c7bb2337cc4b54692213b41bb5647b584ebc5f6c as builder
RUN mkdir /.cache && touch -t 202101010000.00 /.cache

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG TARGETVARIANT=v1

# renovate: datasource=github-releases depName=grpc-ecosystem/grpc-health-probe
ARG GRPC_HEALTH_PROBE_VERSION=v0.4.14
RUN wget -qO/bin/grpc_health_probe "https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-${TARGETOS}-${TARGETARCH}" && \
    chmod +x /bin/grpc_health_probe

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
FROM --platform="${TARGETPLATFORM:-linux/amd64}"  docker.io/alpine:3.16.3@sha256:b95359c2505145f16c6aa384f9cc74eeff78eb36d308ca4fd902eeeb0a0b161b AS runner

USER nobody

COPY --chown=0:0 --from=builder /bin/grpc_health_probe /
COPY --chown=0:0 --from=builder /app/parca /parca
COPY --chown=0:0 parca.yaml /parca.yaml

CMD ["/parca"]
