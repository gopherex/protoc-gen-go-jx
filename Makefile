EXAMPLE_DIR=$(CURDIR)/example
GOLDEN_DIR=$(EXAMPLE_DIR)/golden

.PHONY: build
build:
	go build -o $(CURDIR)/bin/protoc-gen-go-jx ./
	go build -o $(CURDIR)/bin/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go

.PHONY: gen
gen: build
	protoc \
		-I $(GOLDEN_DIR) \
		--plugin=protoc-gen-go=$(CURDIR)/bin/protoc-gen-go \
		--plugin=protoc-gen-go-jx=$(CURDIR)/bin/protoc-gen-go-jx \
		--go_out=$(GOLDEN_DIR) --go_opt=paths=source_relative \
		--go-jx_out=$(GOLDEN_DIR) --go-jx_opt=paths=source_relative \
		$(GOLDEN_DIR)/golden.proto

.PHONY: test
test:
	go clean -testcache && go test ./...

.PHONY: tidy
tidy:
	go mod tidy
