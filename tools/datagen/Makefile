# include $(dir $(lastword $(MAKEFILE_LIST)))/../../config.mk

_PWD     = $(shell pwd | sed "s~/cygdrive/c~c:~")
_APP_DIR = $(_PWD)

APP := datagen

DATAGEN_IMAGE_VERSION   ?= latest
DATAGEN_IMAGE_TAG        = $(AWS_ECR_REGISTRY)/$(IMAGE_PREFIX)$(APP):$(DATAGEN_IMAGE_VERSION)

################################################################################
# Local targets
################################################################################

.PHONY: test
test:
	go test --race ./...

.PHONY: build
build:
	go get ./...
	go build -o bin/datagen cmd/datagen/main.go
	go build -o bin/dynamo_table cmd/dynamo_table/main.go

.PHONY: clean
clean:
	-rm -rf bin/*

.PHONY: image
image: CONTEXT_DIR ?= $(_PWD)
image:
# Adding push to the command to fix pipeline errors
	echo $(CONTEXT_DIR) && echo $(DATAGEN_IMAGE_TAG) && \
    docker build --push --no-cache -t $(DATAGEN_IMAGE_TAG) -f $(CONTEXT_DIR)/Dockerfile --build-arg IMAGE_PREFIX=$(AWS_PUBLIC_IMAGE_PREFIX) --build-arg REGISTRY=$(AWS_PUBLIC_ECR_REGISTRY) $(CONTEXT_DIR)
.PHONY: push
push: CONTEXT_DIR ?= $(_PWD)
push:
	echo $(CONTEXT_DIR) && docker image ls && docker push $(DATAGEN_IMAGE_TAG)

datagen@tag-latest:
	docker tag $(AWS_ECR_REGISTRY)/$(APP):$(DATAGEN_IMAGE_VERSION) $(AWS_ECR_REGISTRY)/$(APP):latest

################################################################################
# Targets run in a containers
################################################################################

DOCKER_SRC_VOLUME  := -v $(_APP_DIR):/usr/src/$(APP)
DOCKER_SRC_WORKDIR := -w /usr/src/$(APP)

GO_IMAGE        := $(AWS_ECR_REGISTRY)/golang:1.16-alpine
GOLANGCI_IMAGE  := $(AWS_ECR_REGISTRY)/golangci-lint:v1.39-alpine

.PHONY: fmt.docker
fmt.docker:
	docker run --rm $(DOCKER_SRC_VOLUME) $(DOCKER_SRC_WORKDIR) $(GO_IMAGE) go mod tidy -v
	docker run --rm $(DOCKER_SRC_VOLUME) $(DOCKER_SRC_WORKDIR) $(GO_IMAGE) gofmt -s -w .

.PHONY: lint.docker
lint.docker:
	docker run --rm $(DOCKER_SRC_VOLUME) $(DOCKER_SRC_WORKDIR) $(GOLANGCI_IMAGE) \
		golangci-lint run -c .golangci.yml ./...

.PHONY: test.docker
test.docker:
	docker run --rm $(DOCKER_SRC_VOLUME) $(DOCKER_SRC_WORKDIR) $(GOLANGCI_IMAGE) go test -race ./...

################################################################################
# Deployment targets
################################################################################

DEPLOYMENT     ?= local
DOCKER_COMPOSE  = docker-compose -f $(_PWD)/deployments/$(DEPLOYMENT)/docker-compose.yaml -p $(APP)

_docker_compose_alias:
	@echo "alias docker-compose=\"$(DOCKER_COMPOSE)\""

_docker_compose_envs:
	@echo DATAGEN_IMAGE_TAG=$(DATAGEN_IMAGE_TAG)

deploy.local:
	DATAGEN_IMAGE_TAG=$(DATAGEN_IMAGE_TAG) $(DOCKER_COMPOSE) up -d --force-recreate
	$(DOCKER_COMPOSE) ps

destroy.local:
	DATAGEN_IMAGE_TAG=$(DATAGEN_IMAGE_TAG) $(DOCKER_COMPOSE) down

################################################################################
# Targets run from repository root
################################################################################

_APP_DIR    = $(_PWD)/tools/datagen

datagen@image:  ## Builds docker image with datagen tool
	$(MAKE) -C $(_APP_DIR) image

datagen@push: ## Pushes docker image with datagen tool
datagen@push: 
	$(MAKE) -C $(_APP_DIR) push

datagen@ci:  ## Run linters and tests for the datagen tool
	$(MAKE) -C $(_APP_DIR) lint.docker test.docker
