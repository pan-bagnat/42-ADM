# syntax=docker/dockerfile:1.7
FROM golang:1.22-alpine AS builder
WORKDIR /app

ENV CGO_ENABLED=0
ENV GOFLAGS="-mod=vendor"
ENV GOCACHE=/go-build-cache

COPY go.mod go.sum ./
COPY vendor ./vendor
COPY cmd ./cmd
COPY internal ./internal

RUN --mount=type=cache,target=/go-build-cache \
    go build -o adm-server ./cmd/server

FROM gcr.io/distroless/base-debian12 AS runner
WORKDIR /app
COPY --from=builder /app/adm-server ./adm-server
EXPOSE 3000
ENTRYPOINT ["/app/adm-server"]
