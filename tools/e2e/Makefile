include $(dir $(lastword $(MAKEFILE_LIST)))/../../config.mk

E2E_IMAGE_VERSION ?= latest
E2E_IMAGE_TAG  = $(AWS_ECR_REGISTRY)/e2e:$(E2E_IMAGE_VERSION)
# If non-empty, use the image of that version as cache source.
E2E_CACHE_TAG ?=

E2E_DOCKER_BUILD_FLAGS = $(and $(E2E_CACHE_TAG),--cache-from=$(AWS_ECR_REGISTRY)/e2e:$(E2E_CACHE_TAG))

################################################################################
# Targets run from repository root
################################################################################

e2e@build: ## Builds docker image with the e2e bidder tests
	docker build -t $(E2E_IMAGE_TAG) -f $(_PWD)/tools/e2e/Dockerfile --build-arg REGISTRY=$(AWS_ECR_REGISTRY) $(E2E_DOCKER_BUILD_FLAGS) $(_PWD)

e2e@push: ## Pushes docker image with the e2e bidder tests
	docker push $(E2E_IMAGE_TAG)

e2e@pull: ## Pulls the image to use as cache source or builds an empty one of such name if the pull fails
	docker pull $(AWS_ECR_REGISTRY)/e2e:$(E2E_CACHE_TAG) || printf 'FROM scratch\nENV X=1\n' | docker build --tag $(AWS_ECR_REGISTRY)/e2e:$(E2E_CACHE_TAG) -

e2e@tag-latest:
	docker tag $(AWS_ECR_REGISTRY)/e2e:$(E2E_IMAGE_VERSION) $(AWS_ECR_REGISTRY)/e2e:latest
