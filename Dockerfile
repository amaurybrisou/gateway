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
    go build \
    -ldflags "-s -w \
    #   -X 'github.com/time-app/backend/pkg/configs.BuildVersion=$BUILD_VERSION' \
    #   -X 'github.com/time-app/backend/pkg/configs.BuildHash=$BUILD_HASH' \
    #   -X 'github.com/time-app/backend/pkg/configs.BuildTime=$BUILD_TIME' \
    " \
    -o ./backend .

FROM --platform=$TARGETPLATFORM alpine:3.17

COPY --from=builder /app/backend /app/backend
CMD ["/app/backend"]
