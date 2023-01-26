include $(dir $(lastword $(MAKEFILE_LIST)))/../../../config.mk

MODEL_APP_ROOT     := ./apps/model
MODEL_APP_VOLUME   := -v $(_PWD)/apps/model:/code
MODEL_GOPATH_CACHE := -v model-go:/go

MODEL_IMAGE_VERSION ?= latest
MODEL_BUILD_OPTIONS ?= ""
# If non-empty, use the image of that version as cache source and use prebuilt images for blackbox tests.

IMAGE_PREFIX=$(STACK_NAME)-

MODEL_DOCKER_BUILD_FLAGS = $(and $(MODEL_CACHE_TAG),--cache-from=$(AWS_ECR_REGISTRY)/model-builder:$(MODEL_CACHE_TAG))
MODEL_IMAGE_TAG ?= $(AWS_ECR_REGISTRY)/$(IMAGE_PREFIX)model:$(MODEL_IMAGE_VERSION)

################################################
# Build docker image
################################################
model@build:
	bash -c "echo $(MODEL_BUILD_OPTIONS)"
	docker build -t $(MODEL_IMAGE_TAG) \
				 -f $(MODEL_APP_ROOT)/tools/Dockerfile.python \
				 --build-arg REGISTRY=$(AWS_PUBLIC_ECR_REGISTRY) \
				 --build-arg IMAGE_PREFIX=$(AWS_PUBLIC_IMAGE_PREFIX) \
				 $(MODEL_DOCKER_BUILD_FLAGS) \
				 $(MODEL_APP_ROOT)

model@build-builder:
	docker build -t $(AWS_ECR_REGISTRY)/model-builder:$(MODEL_IMAGE_VERSION) \
				 -f $(MODEL_APP_ROOT)/tools/Dockerfile.python \
				 --build-arg REGISTRY=$(AWS_ECR_REGISTRY) \
				 --target builder \
				 --build-arg MODEL_BUILD_OPTIONS=$(MODEL_BUILD_OPTIONS) \
				 $(MODEL_DOCKER_BUILD_FLAGS) \
				 $(MODEL_APP_ROOT)

model@pull: ## Pulls the image to use as cache source or builds an empty one of such name if the pull fails
	docker pull $(AWS_ECR_REGISTRY)/model-builder:$(MODEL_CACHE_TAG) || printf 'FROM scratch\nENV X=1\n' | docker build --tag $(AWS_ECR_REGISTRY)/model-builder:$(MODEL_CACHE_TAG) -

################################################
# Push to ECR
################################################
model@push:
	docker push $(MODEL_IMAGE_TAG)

model@push-builder:
	docker push $(AWS_ECR_REGISTRY)/model-builder:$(MODEL_IMAGE_VERSION)

################################################
# Add `latest` tag to the image in version `MODEL_IMAGE_VERSION`
################################################
model@tag-latest:
	docker tag $(AWS_ECR_REGISTRY)/model:$(MODEL_IMAGE_VERSION) $(AWS_ECR_REGISTRY)/model:latest

################################################
# Development
################################################

model@precommit: ## Run precommit checks/lints/formats etc.
model@precommit: model@format model@lint model@test-unit model@test-blackbox

COMPOSE_LOCAL := REGISTRY=$(AWS_ECR_REGISTRY) docker-compose -p local-dev --project-directory ./ \
			-f $(MODEL_APP_ROOT)/tools/docker-compose-local-dev.yaml

model@dev-up: model@dev-up-ls

model@dev-build: ## Build development images
	$(COMPOSE_LOCAL) build

model@dev-up-ls: ## Start model dependencies for local development with Localstack.
	$(COMPOSE_LOCAL) up -d localstack

model@dev-down: ## Stop model dependencies for local development.
	$(COMPOSE_LOCAL) down --remove-orphans

################################################
# Formatting, Linting & mocks
################################################
model@lint: ## Run linters
	$(COMPOSE_LOCAL) run --rm golangci-lint golangci-lint run -c ./tools/.golangci.yml ./...

model@format: ## Run formatters
	docker run --rm $(MODEL_APP_VOLUME) $(MODEL_GOPATH_CACHE) -w /code $(AWS_ECR_REGISTRY)/golang:1.16 go mod tidy -v
	docker run --rm $(MODEL_APP_VOLUME) $(MODEL_GOPATH_CACHE) -w /code $(AWS_ECR_REGISTRY)/golang:1.16 gofmt -s -w .

model@mockery: ## Generate mocks
	docker run -v $(PWD)/apps/model/:/src -w /src vektra/mockery --all --keeptree --recursive --case=underscore --output=tests/mocks

################################################
# Testing
################################################
model@test-unit: ## Run unittests inside docker container
	docker run --rm $(MODEL_APP_VOLUME) $(MODEL_GOPATH_CACHE) -w /code $(AWS_ECR_REGISTRY)/golang:1.16 \
	go test -race ./code/...

model@benchmark: ## Run unittests with benchmarks inside docker container
	docker run --rm $(MODEL_APP_VOLUME) $(MODEL_GOPATH_CACHE) -w /code $(AWS_ECR_REGISTRY)/golang:1.16 \
	go test -bench=. -benchmem ./code/...

COMPOSE_BLACKBOX := REGISTRY=$(AWS_ECR_REGISTRY) MODEL_IMAGE_VERSION=$(MODEL_IMAGE_VERSION) docker-compose -p blackbox-tests --project-directory ./ \
	-f $(MODEL_APP_ROOT)/tools/docker-compose-blackbox.yaml \
	$(if $(MODEL_CACHE_TAG),,-f $(MODEL_APP_ROOT)/tools/docker-compose-blackbox-local.yaml) \

model@_test-blackbox-prepare model@test-blackbox-down:
	$(COMPOSE_BLACKBOX) down --remove-orphans --volumes

model@_test-blackbox-build:
	$(COMPOSE_BLACKBOX) build

model@_test-blackbox-run:
	$(COMPOSE_BLACKBOX) run --rm blackbox-tests go test ./tests/blackbox/...

model@_test-blackbox-logs:
	$(COMPOSE_BLACKBOX) logs model

model@test-blackbox: ## Run blackbox tests inside docker container
model@test-blackbox: model@_test-blackbox-prepare
model@test-blackbox: model@_test-blackbox-build
model@test-blackbox: model@_test-blackbox-run
model@test-blackbox: model@_test-blackbox-logs
model@test-blackbox: model@test-blackbox-down

model@up:  ## Run a model container
	$(COMPOSE_BLACKBOX) up model

model@build-arm64:
	docker buildx build -t $(MODEL_IMAGE_TAG) \
				 -f $(MODEL_APP_ROOT)/tools/Dockerfile.python \
				 --platform linux/amd64,linux/arm64 \
				 --build-arg REGISTRY=$(AWS_ECR_REGISTRY) \
				 --build-arg IMAGE_PREFIX=$(IMAGE_PREFIX) \
				 --push \
				 $(MODEL_DOCKER_BUILD_FLAGS) \
				 $(MODEL_APP_ROOT)
