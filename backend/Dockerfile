# syntax=docker/dockerfile:1

# Stage bersama untuk dev dan produksi supaya keduanya memakai toolchain dan
# dependency yang sama. Hanya go.mod/go.sum yang disalin agar layer ini tetap cache-able.
FROM golang:1.26.5-alpine AS base
WORKDIR /src
ENV CGO_ENABLED=0 GOOS=linux
COPY go.mod go.sum ./
RUN go mod download
RUN GOBIN=/usr/local/bin go install github.com/pressly/goose/v3/cmd/goose@v3.26.0

# Stage dev: source code dibaca dari bind mount host, tidak pernah disalin ke image.
# air mengompilasi ulang otomatis setiap file Go berubah, sehingga mengubah kode tidak
# memerlukan build image baru. Dipakai lewat docker-compose.dev.yml.
FROM base AS dev
ARG TARGET=api
ENV TARGET=${TARGET}
RUN GOBIN=/usr/local/bin go install github.com/air-verse/air@v1.62.0
EXPOSE 8080
ENTRYPOINT []
# tmp_dir diletakkan di luar /src agar artefak build tidak mengotori working tree host.
CMD ["sh", "-c", "exec air --build.cmd \"go build -o /tmp/air/gsnpeeps ./cmd/${TARGET}\" --build.bin /tmp/air/gsnpeeps --build.include_ext go --tmp_dir /tmp/air"]

# Stage build produksi/staging: binary statis untuk api, worker, dan seed.
FROM base AS builder
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
    && go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker \
    && go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

# Stage terakhir adalah default build target, sehingga `docker compose build` tanpa
# target tetap menghasilkan image produksi berbasis binary.
FROM alpine:3.23 AS runtime
ARG TARGET=api
RUN addgroup -S gsnpeeps && adduser -S -G gsnpeeps gsnpeeps
COPY --from=builder /out/${TARGET} /usr/local/bin/gsnpeeps
COPY --from=base /usr/local/bin/goose /usr/local/bin/goose
COPY migrations /migrations
USER gsnpeeps
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gsnpeeps"]
