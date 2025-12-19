# Makefile for the awsme project
# Purpose: Build, lint, test, and manage releases for the awsme project.

REPO_NAME          := awsme
VERSION            ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
YAML_FILES         := $(shell find . -type f \( -iname "*.yml" -o -iname "*.yaml" \) ! -path "./docs/*")
COMMIT             := $(shell git rev-parse HEAD)
BRANCH             := $(shell git rev-parse --abbrev-ref HEAD)
REGISTRY           := gitlab-master.nvidia.com:5005
GO_VERSION	       := $(shell go version | awk '{print $$3}' | sed 's/go//')
GOLINT_VERSION     := $(shell golangci-lint --version | awk '{print $$4}' | sed 's/golangci-lint version //')
KO_VERSION         := $(shell ko version)
GORELEASER_VERSION := $(shell goreleaser --version | sed -n 's/^GitVersion:[[:space:]]*//p')


# Default target
all: help

.PHONY: info
info: ## Prints the current project info
	@echo "version:        $(VERSION)"
	@echo "commit:         $(COMMIT)"
	@echo "branch:         $(BRANCH)"
	@echo "repo:           $(REPO_NAME)"
	@echo "go:             $(GO_VERSION)"
	@echo "linter:         $(GOLINT_VERSION)"
	@echo "ko:             $(KO_VERSION)"
	@echo "goreleaser:     $(GORELEASER_VERSION)"

.PHONY: tidy
tidy: ## Updates Go modules all dependencies
	@set -e; \
	go fmt ./...; \
	go mod tidy

.PHONY: upgrade
upgrade: ## Upgrades all dependencies
	@set -e; \
	go get -u ./...; \
	go mod tidy

.PHONY: lint
lint: lint-go lint-yaml ## Lints the entire project
	@echo "Completed Go and YAML lints"

.PHONY: lint-go
lint-go: ## Lints the Go files
	@set -e; \
	echo "Running golangci-lint"; \
	golangci-lint -c .golangci.yaml run

.PHONY: lint-yaml
lint-yaml: ## Lints YAML files
	@if [ -n "$(YAML_FILES)" ]; then \
		yamllint -c .yamllint.yaml $(YAML_FILES); \
	else \
		echo "No YAML files found to lint."; \
	fi

.PHONY: test
test: ## Runs unit tests
	@set -e; \
	echo "Running tests"; \
	go test -count=1 -race -covermode=atomic -coverprofile=coverage.out ./... || exit 1; \
	echo "Test coverage"; \
	go tool cover -func=coverage.out

.PHONY: scan
scan: ## Scans for source vulnerabilities
	@set -e; \
	echo "Doing static analysis"; \
	go vet ./...; \
	echo "Running vulnerability scan"; \
	grype dir:. --config .grype.yaml --fail-on high --quiet	

.PHONY: qualify
qualify: test lint scan ## Qualifies the current codebase (test, lint, scan)
	@echo "Codebase qualification completed"

.PHONY: build
build: tidy ## Builds the release for the current OS and architecture
	@set -e; \
	KO_DOCKER_REPO=$(REGISTRY) goreleaser build --clean --single-target --snapshot --timeout 10m0s || exit 1; \
	echo "Build completed, binaries are in ./dist"

.PHONY: release
release: ## Runs the release process
	@set -e; \
	goreleaser release --clean --config .goreleaser.yaml --fail-fast --timeout 10m0s

.PHONY: clean
clean: ## Cleans directories
	@set -e; \
	go clean -modcache; \
	go clean ./...; \
	rm -rf ./bin; \
	go get -u ./...; \
	go mod tidy; \
	echo "Cleaned directories"

.PHONY: help
help: ## Displays available commands
	@echo "Available make targets:"; \
	grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk \
		'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'