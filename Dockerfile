ARG GOLANG_BASE
ARG RUNNER_BASE

FROM ${GOLANG_BASE} as builder
RUN mkdir /.cache && chown nobody:nogroup /.cache && touch -t 202101010000.00 /.cache

WORKDIR /app

RUN go install github.com/grpc-ecosystem/grpc-health-probe@latest
# Predicatable path for copying over to final image
RUN if [ "$(go env GOHOSTARCH)" != "$(go env GOARCH)" ]; then mv "$(go env GOPATH)/bin/$(go env GOOS)_$(go env GOARCH)/grpc-health-probe" "$(go env GOPATH)/bin/grpc-health-probe"; fi

ADD dist /app/dist
RUN if [ "amd64" = "$(go env GOARCH)" ]; then cp "dist/parca_$(go env GOOS)_$(go env GOARCH)_$(go env GOAMD64)/parca" parca; else cp "dist/parca_$(go env GOOS)_$(go env GOARCH)/parca" parca; fi

FROM ${RUNNER_BASE} as runner

USER nobody

COPY --chown=0:0 --from=builder /go/bin/grpc-health-probe /
COPY --chown=0:0 --from=builder /app/parca /parca
COPY --chown=0:0 parca.yaml /parca.yaml

CMD ["/parca"]
