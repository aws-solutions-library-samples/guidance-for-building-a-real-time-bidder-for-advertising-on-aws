include $(dir $(lastword $(MAKEFILE_LIST)))/../../config.mk

REPORT_AGGREGATOR_IMAGE_VERSION ?= latest
REPORT_AGGREGATOR_IMAGE_TAG  = $(AWS_ECR_REGISTRY)/report-aggregator:$(REPORT_AGGREGATOR_IMAGE_VERSION)

################################################################################
# Targets run in a containers
################################################################################

RA_DOCKER_SRC_VOLUME  := -v $(_PWD)/tools/report-aggregator:/usr/src/report-aggregator
RA_DOCKER_SRC_WORKDIR := -w /usr/src/report-aggregator

GO_IMAGE        := $(AWS_ECR_REGISTRY)/golang:1.16-alpine
GOLANGCI_IMAGE  := $(AWS_ECR_REGISTRY)/golangci-lint:v1.39-alpine

.PHONY: report-aggregator@fmt
report-aggregator@fmt:
	docker run --rm $(RA_DOCKER_SRC_VOLUME) $(RA_DOCKER_SRC_WORKDIR) $(GO_IMAGE) go mod tidy -v
	docker run --rm $(RA_DOCKER_SRC_VOLUME) $(RA_DOCKER_SRC_WORKDIR) $(GO_IMAGE) gofmt -s -w .

.PHONY: report-aggregator@lint
report-aggregator@lint:
	docker run --rm $(RA_DOCKER_SRC_VOLUME) $(RA_DOCKER_SRC_WORKDIR) $(GOLANGCI_IMAGE) \
		golangci-lint run -c .golangci.yml ./...

################################################################################
# Targets run from repository root
################################################################################

report-aggregator@build: ## Builds docker image with report-aggregator tool
	docker build -t $(REPORT_AGGREGATOR_IMAGE_TAG) --build-arg REGISTRY=$(AWS_ECR_REGISTRY) $(_PWD)/tools/report-aggregator

report-aggregator@push: ## Pushes docker image with report-aggregator tool
	docker push $(REPORT_AGGREGATOR_IMAGE_TAG)

report-aggregator@tag-latest:
	docker tag $(AWS_ECR_REGISTRY)/report-aggregator:$(REPORT_AGGREGATOR_IMAGE_VERSION) $(AWS_ECR_REGISTRY)/report-aggregator:latest

report-aggregator@ci:  ## Run linters and tests for the report-aggregator tool
report-aggregator@ci: report-aggregator@lint
