FROM --platform="${BUILDPLATFORM:-linux/amd64}" docker.io/golang:1.18.4-alpine@sha256:9b2a2799435823dbf6fef077a8ff7ce83ad26b382ddabf22febfc333926400e4 AS builder
RUN mkdir /.cache && touch -t 202101010000.00 /.cache

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG TARGETVARIANT=v1

# renovate: datasource=go depName=github.com/grpc-ecosystem/grpc-health-probe
ARG GRPC_HEALTH_PROBE_VERSION=v0.4.11

RUN go install "github.com/grpc-ecosystem/grpc-health-probe@${GRPC_HEALTH_PROBE_VERSION}"
# Predicatable path for copying over to final image
RUN if [ "$(go env GOHOSTARCH)" != "$(go env GOARCH)" ]; then \
        mv "$(go env GOPATH)/bin/$(go env GOOS)_$(go env GOARCH)/grpc-health-probe" "$(go env GOPATH)/bin/grpc-health-probe"; \
    fi

WORKDIR /app
COPY dist dist

# NOTICE: See goreleaser.yml for the build paths.
RUN if [ "${TARGETARCH}" == 'amd64' ]; then \
        cp "dist/parca_${TARGETOS}_${TARGETARCH}_${TARGETVARIANT:-v1}/parca" . ; \
    elif [ "${TARGETARCH}" == 'arm' ]; then \
        cp "dist/parca_${TARGETOS}_${TARGETARCH}_${TARGETVARIANT##v}/parca" . ; \
    else \
        cp "dist/parca_${TARGETOS}_${TARGETARCH}/parca" . ; \
    fi
RUN chmod +x parca

FROM --platform="${TARGETPLATFORM:-linux/amd64}"  docker.io/alpine:3.16.1@sha256:7580ece7963bfa863801466c0a488f11c86f85d9988051a9f9c68cb27f6b7872 AS runner

USER nobody

COPY --chown=0:0 --from=builder /go/bin/grpc-health-probe /
COPY --chown=0:0 --from=builder /app/parca /parca
COPY --chown=0:0 parca.yaml /parca.yaml

CMD ["/parca"]
