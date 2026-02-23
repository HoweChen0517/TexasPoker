# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS build
WORKDIR /src

ARG TARGETOS=linux
ARG TARGETARCH=amd64

COPY server/go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download || true

COPY server/ ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/pokerd ./cmd/pokerd

FROM gcr.io/distroless/base-debian12
WORKDIR /app

COPY --from=build /out/pokerd /app/pokerd

USER nonroot:nonroot
EXPOSE 8080
ENV POKER_ADDR=:8080

ENTRYPOINT ["/app/pokerd"]
