PROJECT_DIR=$(shell pwd)
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
BIN_DIR=$(PROJECT_DIR)/bin

# BINARY
BINARY_NAME=terraform-provider-edgecenter
TAG_PREFIX="v"
TAG=$(shell git describe --tags)
VERSION=$(shell git describe --tags $(LAST_TAG_COMMIT) | sed "s/^$(TAG_PREFIX)//")
PLUGIN_PATH=~/.terraform.d/plugins/local.edgecenter.ru/repo/edgecenter/$(VERSION)/$(OS)_$(ARCH)

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: build
build: fmtcheck
	mkdir -p $(PLUGIN_PATH)
	go build -o $(PLUGIN_PATH)/$(BINARY_NAME)_v$(VERSION)
	go build -o bin/$(BINARY_NAME)

.PHONY: lint
lint:
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@golangci-lint run -v ./...

.PHONY: test
test:
	go test -v -timeout=2m

fmt:
	gofmt -s -w $(GOFMT_FILES)

.PHONY: fmtcheck
fmtcheck:
	@sh -c "'$(PROJECT_DIR)/scripts/gofmtcheck.sh'"

# DOCS
.PHONY: docs_fmt
docs_fmt:
	terraform fmt -recursive ./examples/

.PHONY: docs
docs: docs_fmt
	go get github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.16
	make tidy
	tfplugindocs --tf-version=1.6.5 --provider-name=edgecenter
