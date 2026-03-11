# FIRST STAGE:  build the app.
FROM docker.io/library/golang:1.25 AS build-app
WORKDIR /build/app

RUN go telemetry off

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

# Dependencies are downloaded only when go.mod or go.sum changes.
RUN go mod download

# Copy the rest of the source files.
COPY . .

RUN make edeps
RUN make statik
RUN make deps

RUN make

# Security scan
RUN go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...

RUN make test

# SECOND STAGE: create the app runtime image.
FROM docker.io/ubuntu:22.04
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates
RUN update-ca-certificates

COPY --from=build-app /build/app/mmsd /app/
WORKDIR /app

RUN chown nobody.nogroup /app
USER nobody:nogroup

ENTRYPOINT ["/app/mmsd"]
