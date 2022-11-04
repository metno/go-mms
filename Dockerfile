# FIRST STAGE:  build the app.
FROM golang:1.16 AS build-app
WORKDIR /build/app

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
RUN make test

# SECOND STAGE: create the app runtime image.
FROM debian:buster-slim
COPY --from=build-app /build/app/mmsd /app/
WORKDIR /app

RUN chown nobody.nogroup /app
USER nobody:nogroup

ENTRYPOINT ["/app/mmsd"]