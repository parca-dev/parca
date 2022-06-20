FROM docker.io/golang:1.18.3-alpine3.15@sha256:f9181168749690bddb6751b004e976bf5d427425e0cfb50522e92c06f761def7 AS builder
RUN mkdir /.cache && chown nobody:nogroup /.cache && touch -t 202101010000.00 /.cache

ARG GRPC_HEALTH_PROBE_VERSION=v0.4.11

WORKDIR /app

RUN go install "github.com/grpc-ecosystem/grpc-health-probe@${GRPC_HEALTH_PROBE_VERSION}"
# Predicatable path for copying over to final image
RUN if [ "$(go env GOHOSTARCH)" != "$(go env GOARCH)" ]; then \
        mv "$(go env GOPATH)/bin/$(go env GOOS)_$(go env GOARCH)/grpc-health-probe" "$(go env GOPATH)/bin/grpc-health-probe"; \
    fi

COPY ./dist /app/dist
RUN if [ "amd64" = "$(go env GOARCH)" ]; then \
        cp "dist/parca_$(go env GOOS)_$(go env GOARCH)_$(go env GOAMD64)/parca" parca; \
    else \
        cp "dist/parca_$(go env GOOS)_$(go env GOARCH)/parca" parca; \
    fi

FROM docker.io/alpine:3.15.4@sha256:4edbd2beb5f78b1014028f4fbb99f3237d9561100b6881aabbf5acce2c4f9454 AS runner

USER nobody

COPY --chown=0:0 --from=builder /go/bin/grpc-health-probe /
COPY --chown=0:0 --from=builder /app/parca /parca
COPY --chown=0:0 parca.yaml /parca.yaml

CMD ["/parca"]
