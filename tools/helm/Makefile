include $(dir $(lastword $(MAKEFILE_LIST)))/../../config.mk

HELM_IMAGE_VERSION ?= latest
HELM_IMAGE_TAG  = $(AWS_ECR_REGISTRY)/helm:$(HELM_IMAGE_VERSION)
# If non-empty, use the image of that version as cache source.
HELM_CACHE_TAG ?=

HELM_DOCKER_BUILD_FLAGS = $(and $(HELM_CACHE_TAG),--cache-from=$(AWS_ECR_REGISTRY)/helm:$(HELM_CACHE_TAG))

################################################################################
# Targets run from repository root
################################################################################

helm@build: ## Builds docker image with the helm and kubectl installed
	docker build -t $(HELM_IMAGE_TAG) -f $(_PWD)/tools/helm/Dockerfile --build-arg REGISTRY=$(AWS_ECR_REGISTRY) $(HELM_DOCKER_BUILD_FLAGS) $(_PWD)

helm@push: ## Pushes docker image with the helm bidder tests
	docker push $(HELM_IMAGE_TAG)

helm@pull: ## Pulls the image to use as cache source or builds an empty one of such name if the pull fails
	docker pull $(AWS_ECR_REGISTRY)/helm:$(HELM_CACHE_TAG) || printf 'FROM scratch\nENV X=1\n' | docker build --tag $(AWS_ECR_REGISTRY)/helm:$(HELM_CACHE_TAG) -

helm@tag-latest:
	docker tag $(AWS_ECR_REGISTRY)/helm:$(HELM_IMAGE_VERSION) $(AWS_ECR_REGISTRY)/helm:latest
