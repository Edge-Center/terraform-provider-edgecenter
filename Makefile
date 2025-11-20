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


create_bin:
	mkdir -p $(BIN_DIR)

install_jq:
	if test "$(OS)" = "linux"; then \
		curl -L -o $(BIN_DIR)/jq https://github.com/stedolan/jq/releases/download/jq-1.7/jq-linux64; \
	elif test "$(ARCH)" = "arm64"; then \
	  	curl -L -o $(BIN_DIR)/jq https://github.com/stedolan/jq/releases/download/jq-1.7/jq-macos-arm64; \
	else \
		curl -L -o $(BIN_DIR)/jq https://github.com/stedolan/jq/releases/download/jq-1.7/jq-osx-amd64; \
	fi
	chmod +x $(BIN_DIR)/jq


install_godotenv:
	go install github.com/joho/godotenv/cmd/godotenv@latest

install_tfplugindocs:
	go get github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.19.4
	make tidy
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.19.4

download_env_file:
	@if [ -z "${VAULT_TOKEN}" ] || [ -z "${VAULT_ADDR}" ]; then \
		echo "ERROR: Vault environment is not set, please setup VAULT_ADDR and VAULT_TOKEN environment variables" && exit 1;\
	fi
	vault kv get -format=json --field data /cloud/terraform | $(BIN_DIR)/jq -r 'to_entries|map("\(.key)=\(.value)")|.[]' > $(ENV_TESTS_FILE)

tidy:
	go mod tidy

init: create_bin install_jq install_godotenv install_tfplugindocs download_env_file tidy
# BUILD
build: tidy
	mkdir -p $(PLUGIN_PATH)
	go build -o $(PLUGIN_PATH)/$(BINARY_NAME)_v$(VERSION)
	go build -o bin/$(BINARY_NAME)

build_debug: tidy
	mkdir -p $(PLUGIN_PATH)
	go build -o $(PLUGIN_PATH)/$(BINARY_NAME)_v$(VERSION) -gcflags '-N -l'
	go build -o bin/$(BINARY_NAME) -gcflags '-N -l'

linters:
	@test -f $(BIN_DIR)/golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.64.8
	@$(BIN_DIR)/golangci-lint run

linters_docker: # for windows
	docker run --rm -v $(PROJECT_DIR):/app -w /app golangci/golangci-lint:v1.64.8 golangci-lint run -v

# TESTS
test_cloud_data_source: install_godotenv
	godotenv -f $(ENV_TESTS_FILE) go test -v $(TEST_DIR) -tags cloud_data_source -short -timeout=60m

test_cloud_resource: install_godotenv
	godotenv -f $(ENV_TESTS_FILE) go test -v $(TEST_DIR) -tags cloud_resource -short -timeout=60m

test_not_cloud: install_godotenv
	godotenv -f $(ENV_TESTS_FILE) go test -v $(TEST_DIR) -tags dns storage cdn -v -timeout=5m

test_cloud_reseller_data_source: install_godotenv
	godotenv -f $(ENV_TESTS_FILE) go test -v $(TEST_DIR) -tags cloud_reseller_data_source -short -timeout=5m

test_cloud_reseller_resource: install_godotenv
	godotenv -f $(ENV_TESTS_FILE) go test -v $(TEST_DIR) -tags cloud_reseller_resource -short -timeout=5m


# DOCS
docs_fmt:
	terraform fmt -recursive ./examples/

docs: docs_fmt
	tfplugindocs --provider-name=edgecenter

.PHONY: tidy build build_debug err_check linters linters_docker envs_reader test_cloud_data_source test_cloud_resource test_not_cloud install_jq install_vault download_env_file test_local_data_source test_local_resource docs_fmt docs
