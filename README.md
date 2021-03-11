# go-mms
![Go](https://github.com/metno/go-mms/workflows/Go/badge.svg?branch=master) 
[![codecov](https://codecov.io/gh/metno/go-mms/branch/master/graph/badge.svg)](https://codecov.io/gh/metno/go-mms)

Go client for MET Messaging system (MMS)

# How to use
## Building

For being able to use `statik` binary, you need to have go's bin directory in the path, most probably `~/go/bin` or `~/.local/go/bin`.

- `cd go-mms`
- `make deps`
- `make`
- `./mmsd`

## Use
See [mms](docs/tldr/mms.md) and [mmsd](docs/tldr/mmsd.md)

## Build and run MMSd as docker container
```
make image
docker run -i -p 4222:4222 -p 8080:8080 mmsd
```

## Generate code diagram
- install go-plantuml: `go get -u github.com/bykof/go-plantuml`
- `make puml`
