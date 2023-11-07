.PHONY: all
all: lint test build

.PHONY: build
build:
	go build -o alerts_exporter .

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	test -z "$$(go fmt ./...)"
	go vet ./...
