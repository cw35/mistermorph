# syntax=docker/dockerfile:1.7

FROM golang:1.24-alpine AS builder
WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/mistermorph ./cmd/mistermorph

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
  && addgroup -S morph \
  && adduser -S -G morph -h /app morph

COPY --from=builder /out/mistermorph /usr/local/bin/mistermorph

RUN mkdir -p /data/state /data/cache /app \
  && chown -R morph:morph /data /app

USER morph
WORKDIR /app

ENV MISTER_MORPH_FILE_STATE_DIR=/data/state \
  MISTER_MORPH_FILE_CACHE_DIR=/data/cache \
  MISTER_MORPH_LOG_FORMAT=json \
  MISTER_MORPH_TOOLS_BASH_ENABLED=false

EXPOSE 8787

ENTRYPOINT ["/usr/local/bin/mistermorph"]
CMD ["serve", "--server-bind", "0.0.0.0", "--server-port", "8787", "--log-level", "info"]
