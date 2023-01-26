# Common configuration

SHELL = /bin/bash
_PWD = $(shell pwd | sed "s~/cygdrive/c~c:~")

ifndef AWS_ACCOUNT
$(error AWS_ACCOUNT environment variable must be set to your AWS account ID)
endif

AWS_REGION       ?= us-east-1
AWS_ECR_REGISTRY := $(AWS_ACCOUNT).dkr.ecr.$(AWS_REGION).amazonaws.com
export AWS_PUBLIC_ECR_REGISTRY = public.ecr.aws/docker/library
export AWS_PUBLIC_IMAGE_PREFIX = 