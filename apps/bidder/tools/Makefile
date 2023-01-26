include $(dir $(lastword $(MAKEFILE_LIST)))/../../../config.mk

BIDDER_APP_ROOT     := ./apps/bidder
BIDDER_APP_VOLUME   := -v $(_PWD)/apps/bidder:/code
BIDDER_GOPATH_CACHE := -v bidder-go:/go

BIDDER_IMAGE_VERSION ?= latest
BIDDER_BUILD_OPTIONS ?= ""
# If non-empty, use the image of that version as cache source and use prebuilt images for blackbox tests.

IMAGE_PREFIX=$(ROOT_STACK)-

BIDDER_DOCKER_BUILD_FLAGS = $(and $(BIDDER_CACHE_TAG),--cache-from=$(AWS_ECR_REGISTRY)/bidder-builder:$(BIDDER_CACHE_TAG))
BIDDER_IMAGE_TAG ?= $(AWS_ECR_REGISTRY)/$(IMAGE_PREFIX)bidder:$(BIDDER_IMAGE_VERSION)

################################################
# Build docker image
################################################
bidder@build:
	bash -c "echo $(BIDDER_BUILD_OPTIONS)"
	docker build -t $(BIDDER_IMAGE_TAG) \
				 -f $(BIDDER_APP_ROOT)/tools/Dockerfile.bidder \
				 --build-arg REGISTRY=$(AWS_PUBLIC_ECR_REGISTRY) \
				 --build-arg IMAGE_PREFIX=$(AWS_PUBLIC_IMAGE_PREFIX) \
				 $(BIDDER_DOCKER_BUILD_FLAGS) \
				 $(BIDDER_APP_ROOT)

bidder@build-builder:
	docker build -t $(AWS_ECR_REGISTRY)/bidder-builder:$(BIDDER_IMAGE_VERSION) \
				 -f $(BIDDER_APP_ROOT)/tools/Dockerfile.bidder \
				 --build-arg REGISTRY=$(AWS_ECR_REGISTRY) \
				 --target builder \
				 --build-arg BIDDER_BUILD_OPTIONS=$(BIDDER_BUILD_OPTIONS) \
				 $(BIDDER_DOCKER_BUILD_FLAGS) \
				 $(BIDDER_APP_ROOT)

bidder@pull: ## Pulls the image to use as cache source or builds an empty one of such name if the pull fails
	docker pull $(AWS_ECR_REGISTRY)/bidder-builder:$(BIDDER_CACHE_TAG) || printf 'FROM scratch\nENV X=1\n' | docker build --tag $(AWS_ECR_REGISTRY)/bidder-builder:$(BIDDER_CACHE_TAG) -

################################################
# Push to ECR
################################################
bidder@push:
	docker push $(BIDDER_IMAGE_TAG)

bidder@push-builder:
	docker push $(AWS_ECR_REGISTRY)/bidder-builder:$(BIDDER_IMAGE_VERSION)

################################################
# Add `latest` tag to the image in version `BIDDER_IMAGE_VERSION`
################################################
bidder@tag-latest:
	docker tag $(AWS_ECR_REGISTRY)/bidder:$(BIDDER_IMAGE_VERSION) $(AWS_ECR_REGISTRY)/bidder:latest

################################################
# Development
################################################

bidder@precommit: ## Run precommit checks/lints/formats etc.
bidder@precommit: bidder@format bidder@lint bidder@test-unit bidder@test-blackbox

COMPOSE_LOCAL := REGISTRY=$(AWS_ECR_REGISTRY) docker-compose -p local-dev --project-directory ./ \
			-f $(BIDDER_APP_ROOT)/tools/docker-compose-local-dev.yaml

bidder@dev-up: bidder@dev-up-ls bidder@dev-up-as ## Start bidder dependencies for local development.
bidder@dev-up: bidder@dev-up-as

bidder@dev-build: ## Build development images
	$(COMPOSE_LOCAL) build

bidder@dev-up-ls: ## Start bidder dependencies for local development with Localstack.
	$(COMPOSE_LOCAL) up -d localstack

bidder@dev-up-as: ## Start bidder dependencies for local development with Aerospike.
	$(COMPOSE_LOCAL) up -d aerospike

bidder@dev-down: ## Stop bidder dependencies for local development.
	$(COMPOSE_LOCAL) down --remove-orphans

bidder@dev-populate:
	$(and $(findstring aerospike,$(DATAGEN_OUTPUT)),$(COMPOSE_LOCAL) up -d aerospike)
	$(and $(findstring aerospike,$(DATAGEN_OUTPUT)),while ! $(COMPOSE_LOCAL) exec aerospike asinfo -v status ; do sleep 1 ; done)
	$(COMPOSE_LOCAL) run --rm datagen -output=$(DATAGEN_OUTPUT) -table=dev -type=devices -max-audience-id=25 -min-audiences=10 -max-audiences=20 -low=1 -high=25
	$(COMPOSE_LOCAL) run --rm datagen -output=$(DATAGEN_OUTPUT) -table=audience_campaigns -type=audiences -max-audience-id=25 -min-audiences=10 -max-audiences=20 -low=1 -high=25
	$(COMPOSE_LOCAL) run --rm datagen -output=$(DATAGEN_OUTPUT) -table=campaign_budget -type=campaigns -low=1 -high=25
	$(COMPOSE_LOCAL) run --rm datagen -output=$(DATAGEN_OUTPUT) -table=budget -operation=clear-set
	$(COMPOSE_LOCAL) run --rm datagen -output=$(DATAGEN_OUTPUT) -table=budget -type=budgets -low=1 -high=25

################################################
# Formatting, Linting & mocks
################################################
bidder@lint: ## Run linters
	$(COMPOSE_LOCAL) run --rm golangci-lint golangci-lint run -c ./tools/.golangci.yml ./...

bidder@format: ## Run formatters
	docker run --rm $(BIDDER_APP_VOLUME) $(BIDDER_GOPATH_CACHE) -w /code $(AWS_ECR_REGISTRY)/golang:1.16 go mod tidy -v
	docker run --rm $(BIDDER_APP_VOLUME) $(BIDDER_GOPATH_CACHE) -w /code $(AWS_ECR_REGISTRY)/golang:1.16 gofmt -s -w .

bidder@mockery: ## Generate mocks
	docker run -v $(PWD)/apps/bidder/:/src -w /src vektra/mockery --all --keeptree --recursive --case=underscore --output=tests/mocks

################################################
# Testing
################################################
bidder@test-unit: ## Run unittests inside docker container
	docker run --rm $(BIDDER_APP_VOLUME) $(BIDDER_GOPATH_CACHE) -w /code $(AWS_ECR_REGISTRY)/golang:1.16 \
	go test -race ./code/...

bidder@benchmark: ## Run unittests with benchmarks inside docker container
	docker run --rm $(BIDDER_APP_VOLUME) $(BIDDER_GOPATH_CACHE) -w /code $(AWS_ECR_REGISTRY)/golang:1.16 \
	go test -bench=. -benchmem ./code/...

COMPOSE_BLACKBOX := REGISTRY=$(AWS_ECR_REGISTRY) BIDDER_IMAGE_VERSION=$(BIDDER_IMAGE_VERSION) docker-compose -p blackbox-tests --project-directory ./ \
	-f $(BIDDER_APP_ROOT)/tools/docker-compose-blackbox.yaml \
	$(if $(BIDDER_CACHE_TAG),,-f $(BIDDER_APP_ROOT)/tools/docker-compose-blackbox-local.yaml) \
	$(and $(findstring aerospike,$(DATAGEN_OUTPUT)),-f $(BIDDER_APP_ROOT)/tools/docker-compose-blackbox-aerospike.yaml)

bidder@_test-blackbox-prepare bidder@test-blackbox-down:
	$(COMPOSE_BLACKBOX) down --remove-orphans --volumes

bidder@_test-blackbox-build:
	$(COMPOSE_BLACKBOX) build

# Set to aerospike to populate Aerospike for the blackbox tests.
DATAGEN_OUTPUT ?= dynamodb

bidder@_test-blackbox-populate:
	$(and $(findstring aerospike,$(DATAGEN_OUTPUT)),$(COMPOSE_BLACKBOX) up -d aerospike)
	$(and $(findstring aerospike,$(DATAGEN_OUTPUT)),while ! $(COMPOSE_BLACKBOX) exec aerospike asinfo -v status ; do sleep 1 ; done)
	$(COMPOSE_BLACKBOX) run --rm datagen -output=$(DATAGEN_OUTPUT) -table=dev -type=devices -max-audience-id=25 -min-audiences=10 -max-audiences=20 -low=1 -high=25
	$(COMPOSE_BLACKBOX) run --rm datagen -output=$(DATAGEN_OUTPUT) -table=audience_campaigns -type=audiences -max-audience-id=25 -min-audiences=10 -max-audiences=20 -low=1 -high=25
	$(COMPOSE_BLACKBOX) run --rm datagen -output=$(DATAGEN_OUTPUT) -table=campaign_budget -type=campaigns -low=1 -high=25
	$(COMPOSE_BLACKBOX) run --rm datagen -output=$(DATAGEN_OUTPUT) -table=budget -operation=clear-set
	$(COMPOSE_BLACKBOX) run --rm datagen -output=$(DATAGEN_OUTPUT) -table=budget -type=budgets -low=1 -high=25

bidder@_test-blackbox-run:
	$(COMPOSE_BLACKBOX) run --rm blackbox-tests go test ./tests/blackbox/...

bidder@_test-blackbox-logs:
	$(COMPOSE_BLACKBOX) logs bidder

bidder@test-blackbox: ## Run blackbox tests inside docker container
bidder@test-blackbox: bidder@_test-blackbox-prepare
bidder@test-blackbox: bidder@_test-blackbox-build
bidder@test-blackbox: bidder@_test-blackbox-populate
bidder@test-blackbox: bidder@_test-blackbox-run
bidder@test-blackbox: bidder@_test-blackbox-logs
bidder@test-blackbox: bidder@test-blackbox-down

bidder@populate-db: ## Populate local database (DynamoDB unless DATAGEN_OUTPUT is set to aerospike)
bidder@populate-db: bidder@_test-blackbox-build bidder@_test-blackbox-populate

bidder@up:  ## Run a bidder container
	$(COMPOSE_BLACKBOX) up bidder

bidder@build-arm64:
	docker buildx build -t $(BIDDER_IMAGE_TAG) \
				 -f $(BIDDER_APP_ROOT)/tools/Dockerfile.bidder \
				 --platform linux/amd64,linux/arm64 \
				 --build-arg REGISTRY=$(AWS_ECR_REGISTRY) \
				 --build-arg IMAGE_PREFIX=$(IMAGE_PREFIX) \
				 --push \
				 $(BIDDER_DOCKER_BUILD_FLAGS) \
				 $(BIDDER_APP_ROOT)
