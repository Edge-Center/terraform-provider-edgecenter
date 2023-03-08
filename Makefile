# ENVS
PROJECT_DIR = $(shell pwd)
BUILD_DIR = $(PROJECT_DIR)/bin
TEST_DIR = $(PROJECT_DIR)/edgecenter/test
ENV_TESTS_FILE = $(TEST_DIR)/.env
export VAULT_ADDR = https://vault.p.ecnl.ru/

# BINARY
BINARY_NAME = terraform-provider-edgecenter
TAG_PREFIX = "v"
TAG = $(shell git describe --tags)
VERSION = $(shell  git describe --tags $(LAST_TAG_COMMIT) | sed "s/^$(TAG_PREFIX)//")
OS := $(shell uname | tr '[:upper:]' '[:lower:]')
ARCH := $(shell uname -m)
PLUGIN_PATH = ~/.terraform.d/plugins/local.edgecenter.ru/repo/edgecenter/$(VERSION)/$(OS)_$(ARCH)

tidy:
	go mod tidy

vendor:
	go mod vendor

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
	@test -f $(BUILD_DIR)/golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.51.1
	@$(BUILD_DIR)/golangci-lint run

# TESTS
envs_reader:
	go install github.com/joho/godotenv/cmd/godotenv@latest

test_cloud_data_source: envs_reader
	godotenv -f $(ENV_TESTS_FILE) go test $(TEST_DIR) -tags cloud_data_source -short -timeout=3m

test_cloud_resource: envs_reader
	godotenv -f $(ENV_TESTS_FILE) go test $(TEST_DIR) -tags cloud_resource -short -timeout=5m

test_not_cloud: envs_reader
	godotenv -f $(ENV_TESTS_FILE) go test $(TEST_DIR) -tags dns storage cdn -v -timeout=5m

# local test run (need to export VAULT_TOKEN env)
jq:
	if test "$(OS)" = "linux"; then \
		curl -L -o jq https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64; \
	else \
		curl -L -o jq https://github.com/stedolan/jq/releases/download/jq-1.6/jq-osx-amd64; \
	fi
	chmod +x jq

vault:
	curl -L -o vault.zip https://releases.hashicorp.com/vault/1.12.3/vault_1.12.3_$(OS)_$(ARCH).zip
	unzip vault.zip && rm -f vault.zip && chmod +x vault

envs:
	vault login -method=token $(VAULT_TOKEN)
	vault kv get -format=json  --field data /CLOUD/terraform | jq -r 'to_entries|map("\(.key)=\(.value)")|.[]' > .local.env

test_local_data_source: envs_reader
	godotenv -f .local.env go test $(TEST_DIR) -tags cloud_data_source -short -timeout=3m -v

test_local_resource: envs_reader
	godotenv -f .local.env go test $(TEST_DIR) -tags cloud_resource -short -timeout=5m -v

# DOCS
docs_fmt:
	terraform fmt -recursive ./examples/

docs: docs_fmt
	go get github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.13.0
	make tidy
	tfplugindocs --tf-version=1.3.8 --website-source-dir=templates

.PHONY: tidy vendor build build_debug err_check linters envs_reader test_cloud test_not_cloud docs_fmt docs
