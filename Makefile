presubmit: test lint
.PHONY: presubmit

test:
	go test -race ./...
.PHONY: test

lint:
	golangci-lint run
.PHONY: lint

build:
	mkdir -p ./bin
	go build -o ./bin/crawl ./main.go
