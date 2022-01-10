# this image is what node:16.6.1-alpine3.14 is on August 12 2021
FROM docker.io/library/node@sha256:456ff86826c47703a7d9b1cbd04b80038e57b86efa4516931148151b379ba035 AS ui-deps

WORKDIR /app

COPY ui/packages/shared ./packages/shared
COPY ui/packages/app/web/package.json ./packages/app/web/package.json
COPY ui/package.json ui/yarn.lock ./
RUN yarn workspace @parca/web install --frozen-lockfile

# Rebuild the source code only when needed
# this image is what node:16.6.1-alpine3.14 is on August 12 2021
FROM docker.io/library/node@sha256:456ff86826c47703a7d9b1cbd04b80038e57b86efa4516931148151b379ba035 AS ui-builder

ENV NODE_ENV production
ENV CIRCLE_NODE_TOTAL 1

WORKDIR /app

COPY ./ui .
COPY --from=ui-deps /app/node_modules ./node_modules
RUN yarn workspace @parca/web build

# this image is what docker.io/golang:1.16.7-alpine3.14 on August 12 2021
FROM docker.io/golang@sha256:7e31a85c5b182e446c9e0e6fba57c522902f281a6a5a6cbd25afa17ac48a6b85 as builder
RUN mkdir /.cache && chown nobody:nogroup /.cache && touch -t 202101010000.00 /.cache

ARG VERSION
ARG COMMIT
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /app

COPY go.mod go.sum /app/
RUN go mod download -modcacherw

COPY --chown=nobody:nogroup go.mod go.sum ./
COPY --chown=nobody:nogroup ./cmd/parca ./cmd/parca
COPY --chown=nobody:nogroup ./pkg ./pkg
COPY --chown=nobody:nogroup ./internal ./internal
COPY --chown=nobody:nogroup ./gen ./gen
COPY --chown=nobody:nogroup ./proto ./proto
COPY --chown=nobody:nogroup ./ui/ui.go ./ui/ui.go
COPY --chown=nobody:nogroup --from=ui-builder /app/packages/app/web/dist ./ui/packages/app/web/dist
RUN go build -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT}" -trimpath -o parca ./cmd/parca
RUN go install github.com/grpc-ecosystem/grpc-health-probe@latest

# this image is what docker.io/alpine:3.14.1 on August 13 2021
FROM docker.io/alpine@sha256:be9bdc0ef8e96dbc428dc189b31e2e3b05523d96d12ed627c37aa2936653258c

USER nobody

COPY --chown=0:0 --from=builder /app/parca /parca
COPY --chown=0:0 --from=builder /go/bin/grpc-health-probe /
COPY --chown=0:0 parca.yaml /parca.yaml

CMD ["/parca"]
