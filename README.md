# go-mms
![Go](https://github.com/metno/go-mms/workflows/Go/badge.svg?branch=master) 
[![codecov](https://codecov.io/gh/metno/go-mms/branch/master/graph/badge.svg)](https://codecov.io/gh/metno/go-mms)

Go client for MET Messaging system (MMS)

# How to use
## Building

To be able to use the `statik` binary, you need to have go's bin directory in the path, most probably `~/go/bin` or `~/.local/go/bin`. 
We found this to be solved by using these paths:

```bash
export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$PATH
export PATH=$PATH:/usr/local/go/bin
```

- `cd go-mms`
- `make edeps`
- `make statik`
- `make deps`
- `make`


## Use
See [mms](docs/tldr/mms.md) and [mmsd](docs/tldr/mmsd.md)

The first step is only relevant for a system where MMS has been installed as a module.

## Build and run MMSd as docker container
```
make image
docker run -i -p 4222:4222 -p 8080:8080 mmsd
```

## Generate code diagram
- install go-plantuml: `go get -u github.com/bykof/go-plantuml`
- `make puml`
