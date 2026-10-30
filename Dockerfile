ARG TARGETPLATFORM

# https://github.com/hadolint/hadolint/issues/861
# hadolint ignore=DL3029
FROM --platform="${BUILDPLATFORM:-linux/amd64}" docker.io/busybox:1.37.0@sha256:a7dbc319b1438115a207e11930dc4bb3e643dcb3ec0bdd694bb36cd8fad289ec as builder
RUN mkdir /.cache && touch -t 202101010000.00 /.cache

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG TARGETVARIANT=v1

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
FROM --platform="${TARGETPLATFORM}" docker.io/alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS runner

LABEL \
    org.opencontainers.image.source="https://github.com/parca-dev/parca" \
    org.opencontainers.image.url="https://github.com/parca-dev/parca" \
    org.opencontainers.image.description="Continuous profiling for analysis of CPU and memory usage, down to the line number and throughout time. Saving infrastructure cost, improving performance, and increasing reliability." \
    org.opencontainers.image.licenses="Apache-2.0"

RUN mkdir /data && chown nobody /data
# hadolint ignore=DL3066
USER nobody

COPY --chown=0:0 --from=builder /app/parca /parca
COPY --chown=0:0 parca.yaml /parca.yaml

CMD ["/parca"]
