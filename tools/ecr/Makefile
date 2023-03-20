include $(dir $(lastword $(MAKEFILE_LIST)))/../../config.mk

ECR_MIRROR_ARCHITECTURES := "amd64 arm64"
ECR_IMAGES_TO_MIRROR := "stelligent/cfn_nag:0.7.12 golang:1.20 golang:1.20-alpine golangci/golangci-lint:v1.39-alpine alpine:3.12.2 aerospike/aerospike:5.5.0.3 python:3.9-slim debian:stretch-slim"
ECR_CODEKIT_IMAGES_TO_MIRROR := "alpine:3.12.2 golang:1.20 golang:1.20-alpine aerospike/aerospike:5.5.0.3 debian:stretch-slim alpine:latest"

ECR_APP_IMAGES := bidder datagen load-generator report-aggregator

ecr@login-public: AWS_REGION=us-east-1
ecr@login-public: ## Log in Docker to the public ECR registry
	aws ecr-public get-login-password --region $(AWS_REGION) | docker login --username AWS --password-stdin $(AWS_PUBLIC_ECR_REGISTRY)

ecr@login: ## Log in Docker to ECR registry
	aws ecr get-login-password --region $(AWS_REGION) | docker login --username AWS --password-stdin $(AWS_ECR_REGISTRY)

ecr@update-mirrors: ## Update ECR mirror repositories
	$(_PWD)/tools/ecr/update-mirror.sh $(IMAGE_PREFIX) $(AWS_ECR_REGISTRY) $(ECR_MIRROR_ARCHITECTURES) $(ECR_IMAGES_TO_MIRROR)

ecr@codekit-update-mirrors: ## Update ECR mirror repositories
	$(_PWD)/tools/ecr/update-mirror.sh $(IMAGE_PREFIX) $(AWS_ECR_REGISTRY) $(ECR_MIRROR_ARCHITECTURES) $(ECR_CODEKIT_IMAGES_TO_MIRROR)

ecr@get-latest-tags:  ## Show concrete tags of the "latest" images of each application repo
	@for repo in $(ECR_APP_IMAGES) ; do \
		echo $$repo && \
		aws ecr describe-images --repository-name $$repo --image-ids imageTag=latest \
			--output json --query "imageDetails[0].imageTags[?@!='latest']" && \
		echo ; \
	done


ecr@deploy-repositories:  ## Create/Update list of repositories on ECR
	aws cloudformation deploy --template-file deployment/infrastructure/ecr.yaml --stack-name ecr  --capabilities CAPABILITY_IAM --no-fail-on-empty-changeset
	aws cloudformation deploy --template-file deployment/infrastructure/ecr-mirror.yaml --stack-name ecr-mirror  --capabilities CAPABILITY_IAM --no-fail-on-empty-changeset
