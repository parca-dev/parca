FROM docker.io/golang:1.18.2-alpine3.15@sha256:e6b729ae22a2f7b6afcc237f7b9da3a27151ecbdcd109f7ab63a42e52e750262 AS builder
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

FROM docker.io/alpine:3.16.0@sha256:686d8c9dfa6f3ccfc8230bc3178d23f84eeaf7e457f36f271ab1acc53015037c AS runner

USER nobody

COPY --chown=0:0 --from=builder /go/bin/grpc-health-probe /
COPY --chown=0:0 --from=builder /app/parca /parca
COPY --chown=0:0 parca.yaml /parca.yaml

CMD ["/parca"]
