# go-mms
Go client for MET Messaging system (MMS)


# How to use
## Build and run in shell
- Clone this repo.

Shell 1.
- `cd go-mms`
- `go build ./cmd/mmsd`
- `./mmsd`

Shell 2:
- `cd go-mms`
- `go build ./cmd/mms`
- `./mms`

You should now get a printout of an MMS message.

## Build and run MMSd as docker container
```bash
cd go-mms
docker build -t mmsd .
docker run -i -p 4222:4222 -p 8080:8080 -p 8088:8088 mmsd
```

## Python Interface

The python interface is meant to be used from [py-mms](https://github.com/metno/py-mms).

Build `libmms.so` with:
```bash
go build -o libmms.so -buildmode=c-shared ./export/
```
