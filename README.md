# go-mms
Go client for MET Messaging system (MMS)


## How to use
- Clone this repo.

Shell 1.
- `cd go-mms`
- `go build ./cmd/rest-api/`
- `./rest-api`

Shell 2:
- `cd go-mms`
- `go build ./cmd/mms`
- `./mms`

You should now get a printout of an MMS message.

## Python Interface

Build `libmms.so` with:
```bash
go build -o libmms.so -buildmode=c-shared ./export/
```
