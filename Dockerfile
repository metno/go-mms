ARG alpine_version=3.23
ARG go_version=1.25

# FIRST STAGE:  build the app.
FROM docker.io/library/golang:${go_version}-alpine${alpine_version} AS build-app
WORKDIR /build/app

RUN --mount=type=cache,target=/var/cache/apk apk add build-base ca-certificates git

RUN go telemetry off

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod go.sum ./

# Dependencies are downloaded only when go.mod or go.sum changes.
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy the rest of the source files.
COPY . .

# Build
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    make deps

# Security scan
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install golang.org/x/vuln/cmd/govulncheck@latest && \
    govulncheck ./... && \
    make && \
    make test


# SECOND STAGE: create the app runtime image.
FROM alpine:${alpine_version}
RUN --mount=type=cache,target=/var/cache/apk apk add ca-certificates curl && update-ca-certificates

COPY --from=build-app /build/app/mmsd /app/

RUN chown nobody:nogroup /app
USER nobody:nogroup

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl --silent -fail http://localhost:8080/api/v1/healthz || exit 1

# Bind not only to localhost, so the app can be accessed from outside the container.
ENTRYPOINT ["/app/mmsd", "--hostname", "0.0.0.0"]
