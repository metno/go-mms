VERSION := $(shell cat ./VERSION)
all: build_mmsd build_mms

build_mmsd: go_mod statik
	go build ./cmd/mmsd

build_mms: go_mod
	go build ./cmd/mms

clean:
	rm mmsd mms

image:
	docker build -t mmsd .

test:
	go test -v ./...

testcov:
	go test -coverprofile=coverage.txt -covermode=atomic -v ./...

statik:
	statik -f -src=static -dest=pkg

go_mod:
	go mod download

edeps:
	go get github.com/rakyll/statik

deps:
	go get -v -t -d ./...

release:
	git tag -a $(VERSION) -m "Release" || true
	git push origin $(VERSION)
	goreleaser --rm-dist

puml:
	go-plantuml generate -rd . -o go-mms.puml

integration_test: build_mmsd
	./mmsd -w ./test_data 2>/dev/null & echo "$$!" > ./mmsd.pid
	go test --tags=integration ./cmd/mms/ || (kill `cat ./mmsd.pid`; unlink ./mmsd.pid; exit 1)

	@kill `cat ./mmsd.pid`
	@unlink ./mmsd.pid

.PHONY: deps go_mod build_mmsd build_mms test image release puml static
