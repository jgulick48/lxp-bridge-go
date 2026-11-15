# set the shell to bash always
SHELL         := /bin/bash

# set make and shell flags to exit on errors
MAKEFLAGS     += --warn-undefined-variables
.SHELLFLAGS   := -euo pipefail -c

# default target is all
.DEFAULT_GOAL := all
.PHONY: all
all: build

# ====================================================================================
# Variables

IMAGE ?= docker.io/jgulick48/lxp-bridge-go
VERSION ?= $(shell date -u +%Y.%m.%d-%H%M)
VERSION := $(VERSION)
GIT_SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
PLATFORMS ?= linux/amd64,linux/arm64
BUILDER ?= lxp-bridge-go
GIT_REMOTE ?= origin
BIN := bin/lxp-bridge-go

# ====================================================================================
# Colors

BLUE         := $(shell printf "\033[34m")
YELLOW       := $(shell printf "\033[33m")
RED          := $(shell printf "\033[31m")
GREEN        := $(shell printf "\033[32m")
CNone        := $(shell printf "\033[0m")

# ====================================================================================
# Logger

TIME_SHORT	= `date +%H:%M:%S`
TIME		= $(TIME_SHORT)

INFO	= echo ${TIME} ${BLUE}[ .. ]${CNone}
OK		= echo ${TIME} ${GREEN}[ OK ]${CNone}
FAIL	= (echo ${TIME} ${RED}[FAIL]${CNone} && false)

# ====================================================================================
# Build Targets

.PHONY: build
build: ## Build the binary
	@$(INFO) go build
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(GIT_SHA)" -o $(BIN) .
	@$(OK) go build

.PHONY: test
test: ## Run tests
	@$(INFO) go test
	go test ./...
	@$(OK) go test

.PHONY: test.race
test.race: ## Run tests with the race detector (requires cgo/gcc)
	@$(INFO) go test -race
	CGO_ENABLED=1 go test -race ./...
	@$(OK) go test -race

.PHONY: vet
vet: ## Run go vet
	@$(INFO) go vet
	go vet ./...
	@$(OK) go vet

.PHONY: fmt
fmt: ## Format code
	@$(INFO) gofmt
	gofmt -w .
	@$(OK) gofmt

.PHONY: tidy
tidy: ## Tidy dependencies
	@$(INFO) go mod tidy
	go mod tidy
	@$(OK) go mod tidy

.PHONY: generate
generate: ## Generate code
	@$(INFO) go generate
	go generate ./...
	@$(OK) go generate

# ====================================================================================
# Docker Targets

.PHONY: docker.buildx.setup
docker.buildx.setup: ## Setup buildx builder
	@$(INFO) docker buildx setup
	@docker buildx inspect $(BUILDER) > /dev/null 2>&1 || docker buildx create --name $(BUILDER) --driver docker-container --bootstrap
	@$(OK) docker buildx setup

.PHONY: docker.build
docker.build: ## Build and load Docker image locally
	@$(INFO) docker build
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(GIT_SHA) -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .
	@$(OK) docker build

.PHONY: git.check-clean
git.check-clean: ## Fail if tracked files have uncommitted changes (the image must match the tagged commit)
	@if ! git diff --quiet HEAD --; then \
		echo "${TIME} ${RED}[FAIL]${CNone} uncommitted changes to tracked files: commit or stash them before docker.push"; exit 1; \
	fi
	@if git rev-parse -q --verify "refs/tags/$(VERSION)" > /dev/null; then \
		echo "${TIME} ${RED}[FAIL]${CNone} git tag $(VERSION) already exists"; exit 1; \
	fi

.PHONY: docker.push
docker.push: git.check-clean docker.buildx.setup ## Build and push Docker image to registry, then tag the commit with the version
	@$(INFO) docker buildx push
	docker buildx build --builder $(BUILDER) --platform $(PLATFORMS) --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(GIT_SHA) -t $(IMAGE):$(VERSION) -t $(IMAGE):latest --push .
	@$(OK) pushed $(IMAGE):$(VERSION)
	@$(MAKE) --no-print-directory git.tag VERSION=$(VERSION)

.PHONY: git.tag
git.tag: ## Tag HEAD with VERSION and push the tag to $(GIT_REMOTE)
	@$(INFO) git tag $(VERSION)
	git tag -a $(VERSION) -m "Docker image $(IMAGE):$(VERSION)" HEAD
	git push $(GIT_REMOTE) refs/tags/$(VERSION)
	@$(OK) tagged $(GIT_SHA) as $(VERSION)

.PHONY: release
release: test docker.push ## Run tests and push Docker image

# ====================================================================================
# Utility Targets

.PHONY: version
version: ## Print version
	@echo $(VERSION)

.PHONY: help
help: ## Display this help message
	@echo -e "$$(grep -hE '^\S+:.*##' $(MAKEFILE_LIST) | sed -e 's/:.*##\s*/:/' -e 's/^\(.\+\):\(.*\)/\\x1b[36m\1\\x1b[m:\2/' | column -c2 -t -s : | sort)"
