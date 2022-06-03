ARG NODE_BUILDER_BASE
ARG GOLANG_BUILDER_BASE
ARG RUNNER_BASE

FROM ${NODE_BUILDER_BASE} AS ui-deps

WORKDIR /app

COPY ui/packages/shared ./packages/shared
COPY ui/packages/app/web/package.json ./packages/app/web/package.json
COPY ui/package.json ui/yarn.lock ./
RUN yarn workspace @parca/web install --frozen-lockfile --network-timeout 100000

# Rebuild the source code only when needed
FROM ${NODE_BUILDER_BASE} AS ui-builder

ENV NODE_ENV production
ENV CIRCLE_NODE_TOTAL 1

WORKDIR /app

COPY ./ui .
COPY --from=ui-deps /app/node_modules ./node_modules
RUN yarn workspace @parca/web build


FROM ${GOLANG_BUILDER_BASE} as builder
RUN mkdir /.cache && chown nobody:nogroup /.cache && touch -t 202101010000.00 /.cache

ARG VERSION
ARG COMMIT
ARG ARCH=amd64
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=$ARCH

WORKDIR /app

COPY go.mod go.sum /app/
RUN go mod download -modcacherw

COPY --chown=nobody:nogroup go.mod go.sum ./
COPY --chown=nobody:nogroup ./cmd/parca ./cmd/parca
COPY --chown=nobody:nogroup ./pkg ./pkg
COPY --chown=nobody:nogroup ./gen ./gen
COPY --chown=nobody:nogroup ./proto ./proto
COPY --chown=nobody:nogroup ./ui/ui.go ./ui/ui.go
COPY --chown=nobody:nogroup --from=ui-builder /app/packages/app/web/build ./ui/packages/app/web/build
RUN go build -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT}" -trimpath -o parca ./cmd/parca
RUN go install github.com/grpc-ecosystem/grpc-health-probe@latest
# Predicatable path for copying over to final image
RUN if [ "$(go env GOHOSTARCH)" != "$(go env GOARCH)" ]; then mv "$(go env GOPATH)/bin/$(go env GOOS)_$(go env GOARCH)/grpc-health-probe" "$(go env GOPATH)/bin/grpc-health-probe"; fi

FROM ${RUNNER_BASE} as runner

USER nobody

COPY --chown=0:0 --from=builder /app/parca /parca
COPY --chown=0:0 --from=builder /go/bin/grpc-health-probe /
COPY --chown=0:0 parca.yaml /parca.yaml

CMD ["/parca"]
