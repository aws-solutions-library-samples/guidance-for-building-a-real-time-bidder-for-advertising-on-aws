include $(dir $(lastword $(MAKEFILE_LIST)))/../../config.mk

LOAD_GENERATOR_IMAGE_VERSION   ?= latest
LOAD_GENERATOR_IMAGE_TAG       ?= $(AWS_ECR_REGISTRY)/$(IMAGE_PREFIX)load-generator:$(LOAD_GENERATOR_IMAGE_VERSION)

################################################################################
# Targets run in a containers
################################################################################

LG_DOCKER_SRC_VOLUME  := -v $(_PWD)/tools/load-generator:/usr/src/load-generator
LG_DOCKER_SRC_WORKDIR := -w /usr/src/load-generator

GO_IMAGE        := $(AWS_ECR_REGISTRY)/golang:1.16-alpine
GOLANGCI_IMAGE  := $(AWS_ECR_REGISTRY)/$(IMAGE_PREFIX)golangci-lint:v1.39-alpine

.PHONY: load-generator@precommit
load-generator@precommit: load-generator@fmt load-generator@lint load-generator@test ## Run precommit checks/lints/formats etc.

.PHONY: load-generator@fmt
load-generator@fmt: ## Run formatters
	docker run --rm $(LG_DOCKER_SRC_VOLUME) $(LG_DOCKER_SRC_WORKDIR) $(GO_IMAGE) go mod tidy -v
	docker run --rm $(LG_DOCKER_SRC_VOLUME) $(LG_DOCKER_SRC_WORKDIR) $(GO_IMAGE) gofmt -s -w .

.PHONY: load-generator@lint
load-generator@lint: ## Run linters
	docker run --rm $(LG_DOCKER_SRC_VOLUME) $(LG_DOCKER_SRC_WORKDIR) $(GOLANGCI_IMAGE) \
		golangci-lint run -c .golangci.yml ./...

.PHONY: load-generator@test
load-generator@test: ## Run unit tests
	docker run --rm $(LG_DOCKER_SRC_VOLUME) $(LG_DOCKER_SRC_WORKDIR) $(GOLANGCI_IMAGE) go test -race ./...

################################################################################
# Targets run from repository root
################################################################################

load-generator@build: ## Builds docker image with load-generator tool, Adding push to the command to fix pipeline errors
	docker build --push -t $(LOAD_GENERATOR_IMAGE_TAG) --build-arg IMAGE_PREFIX=$(AWS_PUBLIC_IMAGE_PREFIX) --build-arg REGISTRY=$(AWS_PUBLIC_ECR_REGISTRY) $(_PWD)/tools/load-generator
load-generator@push: ## Pushes docker image with load-generator tool
	docker push $(LOAD_GENERATOR_IMAGE_TAG)

load-generator@tag-latest:
	docker tag $(AWS_ECR_REGISTRY)/${IMAGE_PREFIX}load-generator:$(LOAD_GENERATOR_IMAGE_VERSION) $(AWS_ECR_REGISTRY)/load-generator:latest

load-generator@ci:  ## Run linters and tests for the load-generator tool
load-generator@ci: load-generator@lint load-generator@test
