# go-mms
![Go](https://github.com/metno/go-mms/workflows/Go/badge.svg?branch=master) 
[![codecov](https://codecov.io/gh/metno/go-mms/branch/master/graph/badge.svg)](https://codecov.io/gh/metno/go-mms)

Go client for MET Messaging system (MMS)

# How to use
## Build and run in shell

For being able to use `statik` binary, you need to have go's bin directory in the path, most probably `~/go/bin` or `~/.local/go/bin`.

Shell 1:
- `cd go-mms`
- `make deps`
- `make`
- `./mmsd`

Shell 2:
- `./mms s`

Shell 3:
- `./mms p --production-hub test-hub --product test`

You should now get a printout of an MMS message in `Shell 2`

## Build and run MMSd as docker container
```
make image
docker run -i -p 4222:4222 -p 8080:8080 mmsd
```

## Generate code diagram
- install go-plantuml: `go get -u github.com/bykof/go-plantuml`
- `make puml`
