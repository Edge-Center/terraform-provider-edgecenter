# ENVS
ifeq ($(OS),Windows_NT)
	PROJECT_DIR = $(shell cd)
	OS := windows
	ifeq ($(PROCESSOR_ARCHITECTURE),AMD64)
		ARCH := amd64
	endif
	ifeq ($(PROCESSOR_ARCHITECTURE),x86)
		ARCH := 386
	endif
else
	PROJECT_DIR = $(shell pwd)
	OS := $(shell uname | tr '[:upper:]' '[:lower:]')
    ARCH := $(shell uname -m)
endif
BIN_DIR = $(PROJECT_DIR)/bin
TEST_DIR = $(PROJECT_DIR)/edgecenter/test
ENV_TESTS_FILE = $(TEST_DIR)/.env

# BINARY
BINARY_NAME = terraform-provider-edgecenter
TAG_PREFIX = "v"
TAG = $(shell git describe --tags)
VERSION = $(shell git describe --tags $(LAST_TAG_COMMIT) | sed "s/^$(TAG_PREFIX)//")
PLUGIN_PATH = ~/.terraform.d/plugins/local.edgecenter.ru/repo/edgecenter/$(VERSION)/$(OS)_$(ARCH)

tidy:
	go mod tidy

# BUILD
build: tidy
	mkdir -p $(PLUGIN_PATH)
	go build -o $(PLUGIN_PATH)/$(BINARY_NAME)_v$(VERSION)
	go build -o bin/$(BINARY_NAME)

build_debug: tidy
	mkdir -p $(PLUGIN_PATH)
	go build -o $(PLUGIN_PATH)/$(BINARY_NAME)_v$(VERSION) -gcflags '-N -l'
	go build -o bin/$(BINARY_NAME) -gcflags '-N -l'

# CHECKS
err_check:
	@sh -c "'$(PROJECT_DIR)/scripts/errcheck.sh'"

linters:
	@test -f $(BIN_DIR)/golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.54.2
	@$(BIN_DIR)/golangci-lint run

linters_docker: # for windows
	docker run --rm -v $(PROJECT_DIR):/app -w /app golangci/golangci-lint:v1.54.2 golangci-lint run -v

# TESTS
envs_reader:
	go install github.com/joho/godotenv/cmd/godotenv@latest

test_cloud_data_source: envs_reader
	godotenv -f $(ENV_TESTS_FILE) go test $(TEST_DIR) -tags cloud_data_source -short -timeout=20m

test_cloud_resource: envs_reader
	godotenv -f $(ENV_TESTS_FILE) go test $(TEST_DIR) -tags cloud_resource -short -timeout=20m

test_not_cloud: envs_reader
	godotenv -f $(ENV_TESTS_FILE) go test $(TEST_DIR) -tags dns storage cdn -v -timeout=5m

# local test run (need to export VAULT_TOKEN env)
install_jq:
	if test "$(OS)" = "linux"; then \
		curl -L -o $(BIN_DIR)/jq https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64; \
	else \
		curl -L -o $(BIN_DIR)/jq https://github.com/stedolan/jq/releases/download/jq-1.6/jq-osx-amd64; \
	fi
	chmod +x $(BIN_DIR)/jq

install_vault:
	curl -L -o vault.zip https://releases.hashicorp.com/vault/1.13.3/vault_1.13.3_$(OS)_$(ARCH).zip
	unzip vault.zip && rm -f vault.zip && chmod +x vault
	mv vault $(BIN_DIR)/

download_env_file: envs_reader
	godotenv -f $(ENV_TESTS_FILE) $(BIN_DIR)/vault login -method=token $(VAULT_TOKEN)
	godotenv -f $(ENV_TESTS_FILE) $(BIN_DIR)/vault kv get -format=json --field data /CLOUD/terraform | $(BIN_DIR)/jq -r 'to_entries|map("\(.key)=\(.value)")|.[]' >> $(ENV_TESTS_FILE)

test_local_data_source: envs_reader
	godotenv -f .local.env go test $(TEST_DIR) -tags cloud_data_source -short -timeout=5m -v

test_local_resource: envs_reader
	godotenv -f .local.env go test $(TEST_DIR) -tags cloud_resource -short -timeout=10m -v

# DOCS
docs_fmt:
	terraform fmt -recursive ./examples/

docs: docs_fmt
	go get github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.16
	make tidy
	tfplugindocs --tf-version=1.5.0 --provider-name=edgecenter

.PHONY: tidy build build_debug err_check linters linters_docker envs_reader test_cloud_data_source test_cloud_resource test_not_cloud install_jq install_vault download_env_file test_local_data_source test_local_resource docs_fmt docs
