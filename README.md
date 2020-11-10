# go-mms
![Go](https://github.com/metno/go-mms/workflows/Go/badge.svg?branch=master) 
[![codecov](https://codecov.io/gh/metno/go-mms/branch/master/graph/badge.svg)](https://codecov.io/gh/metno/go-mms)

Go client for MET Messaging system (MMS)

# How to use
## Build and run in shell

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
cd go-mms
docker build -t mmsd .
docker run -i -p 4222:4222 -p 8080:8080 mmsd
```

## Python Interface

The python interface is meant to be used from [py-mms](https://github.com/metno/py-mms).

Build `libmms.so` with:
```bash
go build -o libmms.so -buildmode=c-shared ./export/
```

## Generate code diagram
- install go-plantuml: `go get -u github.com/bykof/go-plantuml`
- `make puml`
