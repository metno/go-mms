VERSION := $(shell cat ./VERSION)
all: build_mmsd build_mms

build_mmsd:
	go build ./cmd/mmsd

build_mms:
	go build ./cmd/mms

clean:
	rm mmsd mms

image:
	docker build -t mmsd .

test:
	cd ./lib && go test -v


release:
	git tag -a $(VERSION) -m "Release" || true
	git push origin $(VERSION)
	goreleaser --rm-dist

puml:
	go-plantuml generate -rd . -o go-mms.puml

.PHONY: build_mmsd build_mms test image release puml
