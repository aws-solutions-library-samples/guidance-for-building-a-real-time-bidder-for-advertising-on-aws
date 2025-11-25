#!/usr/bin/env bash
set -ex
# Check if jq is available
type jq >/dev/null 2>&1 || { echo >&2 "The jq utility is required for this script to run."; exit 1; }

# Check if aws cli is available
type aws >/dev/null 2>&1 || { echo >&2 "The aws cli is required for this script to run."; exit 1; }

export SUPPORTED_AWS_REGIONS="us\-east\-1|us\-west\-1|us\-west\-2|us\-east\-2"

#if [[ -z "${AWS_ACCOUNT}" ]]; then
#    echo "AWS Account Id [0-9]:"
#    read AWS_ACCOUNT
#fi

export AWS_ACCOUNT=$1

#if [[ -z "${AWS_REGION}" ]]; then
#    echo "AWS Region:"
#    read AWS_REGION
#fi

export AWS_REGION=$2

if ! sh -c "echo $AWS_REGION | grep -q -E '^(${SUPPORTED_AWS_REGIONS})$'" ; then
    echo "Unsupported AWS region: ${AWS_REGION}"
    exit 1
fi

#if [[ -z "${STACK_NAME}" ]]; then
#    echo "Stack name [a-z0-9]:"
#    read STACK_NAME
#fi

export STACK_NAME=$3

make ecr@login

# Check if ECR repository exists, create if not
REPO_NAME="${STACK_NAME}-load-generator"
if ! aws ecr describe-repositories --repository-names ${REPO_NAME} --region ${AWS_REGION} >/dev/null 2>&1; then
  echo "Creating ECR repository: ${REPO_NAME}"
  aws ecr create-repository --repository-name ${REPO_NAME} --region ${AWS_REGION}
fi


make load-generator@build
    
echo "[Setup] The bidder loadgen has been deployed."
