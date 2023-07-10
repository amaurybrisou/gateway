FROM --platform=$BUILDPLATFORM golang:1.20.4-alpine3.17 as builder
SHELL ["/bin/ash", "-o", "pipefail", "-c"]
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./

ARG BUILD_VERSION
ARG BUILD_HASH
ARG BUILD_TIME

RUN export "GOOS=$(echo "$TARGETPLATFORM" | cut -d/ -f1)"; \
    export "GOARCH=$(echo "$TARGETPLATFORM" | cut -d/ -f2)"; \
    export CGO_ENABLED=0; \
    ENV=production go build \
    -ldflags "-s -w \
    -X 'github.com/amaurybrisou/gateway/src.BuildHash=$BUILD_VERSION' \
    -X 'github.com/amaurybrisou/gateway/src.BuildHash=$BUILD_HASH' \
    -X 'github.com/amaurybrisou/gateway/src.BuildTime=$BUILD_TIME' \
    " \
    -o ./backend cmd/gateway/main.go

FROM --platform=$TARGETPLATFORM scratch

COPY --from=builder /app/backend /app/backend
COPY --from=builder /app/migrations /app/migrations
CMD ["/app/backend"]
