# FIRST STAGE:  build the app.
FROM golang:1.15 AS build-app
WORKDIR /build/app

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

# Dependencies are downloaded only when go.mod or go.sum changes.
RUN go mod download

# Copy the rest of the source files.
COPY . .
<<<<<<< HEAD
RUN go test ./...
=======
RUN go test ./pkg/mms
>>>>>>> master
RUN go build -o mmsd ./cmd/mmsd

# SECOND STAGE: create the app runtime image.
FROM ubuntu:bionic

COPY --from=build-app /build/app/mmsd /app/
COPY --from=build-app /build/app/static /app/static
COPY --from=build-app /build/app/templates /app/templates

WORKDIR /app
USER nobody:nogroup

ENTRYPOINT ["/app/mmsd"]
