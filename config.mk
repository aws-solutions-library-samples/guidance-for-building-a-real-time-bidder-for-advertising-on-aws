# Common configuration
-include .env
export $(shell if [ -f .env ]; then sed 's/=.*//' .env; fi)
SHELL = /bin/bash
_PWD = $(shell pwd | sed "s~/cygdrive/c~c:~")
AWS_ACCOUNT = $(shell aws sts get-caller-identity --query Account --output text)
ifndef AWS_ACCOUNT
$(error AWS_ACCOUNT environment variable must be set to your AWS account ID)
endif

export AWS_REGION       ?= us-east-1
export AWS_ECR_REGISTRY := $(AWS_ACCOUNT).dkr.ecr.$(AWS_REGION).amazonaws.com
ifndef STACK_NAME
export STACK_NAME := rtb-$(AWS_ACCOUNT)
endif
export ROOT_STACK_NAME := $(STACK_NAME)

export AWS_PUBLIC_ECR_REGISTRY = public.ecr.aws/docker/library
export AWS_PUBLIC_IMAGE_PREFIX = 

