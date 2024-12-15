# set the shell to bash always
SHELL         := /bin/bash

# set make and shell flags to exit on errors
MAKEFLAGS     += --warn-undefined-variables
.SHELLFLAGS   := -euo pipefail -c

ARCH = amd64
BUILD_ARGS ?=

DOCKER_BUILD_PLATFORMS = linux/amd64,linux/arm64
DOCKER_BUILDX_BUILDER ?= "cluster-config-maps"

# default target is build
.DEFAULT_GOAL := all
.PHONY: all
all: $(addprefix build-,$(ARCH))

# Image registry for build/push image targets
IMAGE_REGISTRY ?= ghcr.io/indeedeng/cluster-config-maps

CRD_OPTIONS ?= "crd"
CRD_DIR     ?= deploy/crds

HELM_DIR    ?= deploy/charts/cluster-config-maps

OUTPUT_DIR  ?= bin

RUN_GOLANGCI_LINT := go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1

# check if there are any existing `git tag` values
ifeq ($(shell git tag),)
# no tags found - default to initial tag `v0.0.0`
VERSION ?= $(shell echo "v0.0.0-$$(git rev-list HEAD --count)-g$$(git describe --dirty --always)" | sed 's/-/./2' | sed 's/-/./2')
else
# use tags
VERSION ?= $(shell git describe --dirty --always --tags --exclude 'helm*' | sed 's/-/./2' | sed 's/-/./2')
endif

# ====================================================================================
# Colors

BLUE         := $(shell printf "\033[34m")
YELLOW       := $(shell printf "\033[33m")
RED          := $(shell printf "\033[31m")
GREEN        := $(shell printf "\033[32m")
CNone        := $(shell printf "\033[0m")

# ====================================================================================
# Logger

TIME_LONG	= `date +%Y-%m-%d' '%H:%M:%S`
TIME_SHORT	= `date +%H:%M:%S`
TIME		= $(TIME_SHORT)

INFO	= echo ${TIME} ${BLUE}[ .. ]${CNone}
WARN	= echo ${TIME} ${YELLOW}[WARN]${CNone}
ERR		= echo ${TIME} ${RED}[FAIL]${CNone}
OK		= echo ${TIME} ${GREEN}[ OK ]${CNone}
FAIL	= (echo ${TIME} ${RED}[FAIL]${CNone} && false)

# ====================================================================================
# Conformance

# Ensure a PR is ready for review.
reviewable: generate helm.generate
	@go mod tidy

# Ensure branch is clean.
check-diff: reviewable
	@$(INFO) checking that branch is clean
	@test -z "$$(git status --porcelain)" || (echo "$$(git status --porcelain)" && $(FAIL))
	@$(OK) branch is clean

# ====================================================================================
# Golang

install_tools:
	go install golang.org/x/tools/cmd/stringer

.PHONY: test
test: generate lint ## Run tests
	@$(INFO) go test unit-tests
	go test -race -v ./... -coverprofile cover.out
	@$(OK) go test unit-tests

.PHONY: build
build: $(addprefix build-,$(ARCH))

.PHONY: build-%
build-%: generate ## Build binary for the specified arch
	@$(INFO) go build $*
	@CGO_ENABLED=0 GOOS=linux GOARCH=$* \
		go build -o '$(OUTPUT_DIR)/ccm-csi-plugin-$*' ./cmd/ccm-csi-plugin/main.go
	@$(OK) go build $*

.PHONY: lint
lint: ## run golangci-lint
	$(RUN_GOLANGCI_LINT) run

fmt: ## ensure consistent code style
	@go mod tidy
	@go fmt ./...
	$(RUN_GOLANGCI_LINT) run --fix > /dev/null 2>&1 || true
	@$(OK) Ensured consistent code style

generate: ## Generate code and crds
	@go generate
	@$(OK) Finished generating deepcopy and crds

# ====================================================================================
# Documentation
.PHONY: docs
docs: generate
	$(MAKE) -C ./hack/api-docs build

.PHONY: serve-docs
serve-docs:
	$(MAKE) -C ./hack/api-docs serve

docker.build: docker.buildx.setup ## Build the docker image
	@$(INFO) docker build
	@docker buildx build --platform $(DOCKER_BUILD_PLATFORMS) -t $(IMAGE_REGISTRY):$(VERSION) $(BUILD_ARGS) --push .
	@$(OK) docker build

docker.buildx.setup:
	@$(INFO) docker buildx setup
	@docker buildx ls 2>/dev/null | grep -vq $(DOCKER_BUILDX_BUILDER) || docker buildx create --name $(DOCKER_BUILDX_BUILDER) --driver docker-container --driver-opt network=host --bootstrap --use
	@$(OK) docker buildx setup

# ====================================================================================
# Help

# only comments after make target name are shown as help text
help: ## displays this help message
	@echo -e "$$(grep -hE '^\S+:.*##' $(MAKEFILE_LIST) | sed -e 's/:.*##\s*/:/' -e 's/^\(.\+\):\(.*\)/\\x1b[36m\1\\x1b[m:\2/' | column -c2 -t -s : | sort)"
