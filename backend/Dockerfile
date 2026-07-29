FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
RUN GOBIN=/out go install github.com/pressly/goose/v3/cmd/goose@v3.26.0
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

FROM alpine:3.23
ARG TARGET=api
RUN addgroup -S gsnpeeps && adduser -S -G gsnpeeps gsnpeeps
COPY --from=builder /out/${TARGET} /usr/local/bin/gsnpeeps
COPY --from=builder /out/goose /usr/local/bin/goose
COPY migrations /migrations
USER gsnpeeps
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gsnpeeps"]
