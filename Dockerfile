FROM --platform="${BUILDPLATFORM:-linux/amd64}" docker.io/golang:1.18.4-alpine@sha256:53bdbdf45bfab272f7c9897ba57dc34be258a210a8cbc5dfb5ed6312c95182e6 AS builder
RUN mkdir /.cache && touch -t 202101010000.00 /.cache

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG TARGETVARIANT

# renovate: datasource=go depName=github.com/grpc-ecosystem/grpc-health-probe
ARG GRPC_HEALTH_PROBE_VERSION=v0.4.11

RUN go install "github.com/grpc-ecosystem/grpc-health-probe@${GRPC_HEALTH_PROBE_VERSION}"
# Predicatable path for copying over to final image
RUN if [ "$(go env GOHOSTARCH)" != "$(go env GOARCH)" ]; then \
        mv "$(go env GOPATH)/bin/$(go env GOOS)_$(go env GOARCH)/grpc-health-probe" "$(go env GOPATH)/bin/grpc-health-probe"; \
    fi

WORKDIR /app
COPY ./dist /app/dist

RUN if [ "amd64" = "$(go env GOARCH)" ]; then \
        cp "dist/parca_$(go env GOOS)_$(go env GOARCH)_$(go env GOAMD64)/parca" parca; \
    else \
        cp "dist/parca_$(go env GOOS)_$(go env GOARCH)/parca" parca; \
    fi

RUN chmod +x parca

FROM --platform="${TARGETPLATFORM:-linux/amd64}"  docker.io/alpine:3.16.1@sha256:7580ece7963bfa863801466c0a488f11c86f85d9988051a9f9c68cb27f6b7872 AS runner

USER nobody

COPY --chown=0:0 --from=builder /go/bin/grpc-health-probe /
COPY --chown=0:0 --from=builder /app/parca /parca
COPY --chown=0:0 parca.yaml /parca.yaml

CMD ["/parca"]
