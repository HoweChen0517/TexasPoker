# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS build
WORKDIR /app

COPY server/go.mod ./
RUN go mod download || true

COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/pokerd ./cmd/pokerd

FROM gcr.io/distroless/base-debian12
WORKDIR /app

COPY --from=build /out/pokerd /app/pokerd

EXPOSE 8080
ENV POKER_ADDR=:8080

CMD ["/app/pokerd"]
