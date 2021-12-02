all: build_mmsd build_mms

build_mmsd: go_mod statik
	go build ./cmd/mmsd

build_mms: go_mod
	go build ./cmd/mms

clean:
	rm mmsd mms

image:
	docker build -t mmsd .

testdb:
	cp test/state.db.static test/state.db

test:
	go test -v ./...

testcov:
	go test -coverprofile=coverage.txt -covermode=atomic -v ./...

statik:
	statik -f -src=static -dest=pkg

go_mod:
	go mod download

edeps:
	go install github.com/rakyll/statik

deps:
	go get -v -t -d ./...

release:
	git tag -a $(VERSION) -m "Release" || true
	git push origin $(VERSION)
	goreleaser --rm-dist

puml:
	go-plantuml generate -rd . -o go-mms.puml

integration_test: build_mmsd testdb
	./mmsd -w ./test 2>/dev/null & echo "$$!" > ./mmsd.pid
	go test -count=1 --tags=integration ./cmd/mms/ || (kill `cat ./mmsd.pid`; unlink ./mmsd.pid; exit 1)

	@kill `cat ./mmsd.pid`
	@unlink ./mmsd.pid

.PHONY: deps go_mod build_mmsd build_mms test image release puml static
