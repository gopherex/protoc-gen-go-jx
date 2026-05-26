EXAMPLE_DIR=$(CURDIR)/example

.PHONY: build
build:
	go build -o $(CURDIR)/bin/protoc-gen-go-jx ./

.PHONY: gen
gen: build
	cd $(EXAMPLE_DIR) && easyp generate

.PHONY: test
test:
	go clean -testcache && go test ./...

.PHONY: tidy
tidy:
	go mod tidy
