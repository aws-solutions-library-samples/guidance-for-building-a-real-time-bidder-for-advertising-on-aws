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

#if [[ -z "${VARIANT}" ]]; then
#    echo "Database variant (DynamoDB):"
#    read VARIANT
#fi
export VARIANT=$4

#echo "Populate the database with test data (yes|no):"
#read USE_DATAGEN

export USE_DATAGEN=$5

if ! sh -c "echo $VARIANT | grep -q -E '^(DynamoDB|Aerospike)$'" ; then
    echo "Unsupported database variant: ${VARIANT}"
    exit 1
fi

#echo "Deploy the load generator (yes|no):"
#read USE_LOAD_GENERATOR

export USE_LOAD_GENERATOR=$6

if ! sh -c "echo $USE_LOAD_GENERATOR | grep -q -E '^(yes|no)$'" ; then
    echo "Invalid input: ${USE_LOAD_GENERATOR} instead of (yes|no)"
    exit 1
fi


if ! sh -c "echo $USE_DATAGEN | grep -q -E '^(yes|no)$'" ; then
    echo "Invalid input: ${USE_DATAGEN} instead of (yes|no)"
    exit 1
fi

export BIDDER_IMAGE_REPOSITORY=${STACK_NAME}-bidder
export IMAGE_PREFIX="${STACK_NAME}-"

export CF_TEMP_DIR=`mktemp -d`
export CF_TEMP_FILE=`mktemp -p ${CF_TEMP_DIR}`

touch ${CF_TEMP_FILE}

export CF_BUCKET_NAME=${STACK_NAME}-cf-templates
echo "[Setup] Creating a S3 bucket (${CF_BUCKET_NAME}) to store the Cloudformation stack package..."

if aws s3api head-bucket --bucket ${CF_BUCKET_NAME} --region ${AWS_REGION} 2>&1 | grep -q 'Not Found'; then
    if [ "${AWS_REGION}" != "us-east-1" ]; then
        LOCATION_CONSTRAINT="--create-bucket-configuration LocationConstraint=${AWS_REGION}"
    fi
    aws s3api create-bucket --bucket ${CF_BUCKET_NAME} --region ${AWS_REGION} ${LOCATION_CONSTRAINT}
    aws s3api put-bucket-encryption \
    --bucket ${CF_BUCKET_NAME} \
    --server-side-encryption-configuration '{"Rules": [{"ApplyServerSideEncryptionByDefault": {"SSEAlgorithm": "AES256"}}]}'
fi

echo "[Setup] Deploying the Cloudformation stack..."
aws cloudformation package \
    --template-file $(pwd)/deployment/infrastructure/codekit.yaml \
    --output-template-file ${CF_TEMP_FILE} \
    --s3-bucket ${CF_BUCKET_NAME}

aws cloudformation deploy \
    --template-file ${CF_TEMP_FILE} \
    --stack-name "${STACK_NAME}" \
    --capabilities CAPABILITY_NAMED_IAM \
    --parameter-overrides \
        "ProjectName=${STACK_NAME}" \
        "Variant=Codekit${VARIANT}" \
    --no-fail-on-empty-changeset


export APPLICATION_STACK_NAME=`aws cloudformation describe-stacks --stack-name ${STACK_NAME} --output json | jq '.Stacks[].Outputs[] | select(.OutputKey=="ApplicationStackARN") | .OutputValue' | cut -d/ -f2`
export CODEBUILD_STACK_NAME=`aws cloudformation describe-stacks --stack-name ${STACK_NAME} --output json | jq '.Stacks[].Outputs[] | select(.OutputKey=="CodebuildStackARN") | .OutputValue' | cut -d/ -f2`
export EKS_WORKER_ROLE_ARN=`aws cloudformation describe-stacks --stack-name ${STACK_NAME} --output json | jq -r '.Stacks[].Outputs[] | select(.OutputKey=="EKSWorkerRoleARN") | .OutputValue'`
export EKS_ACCESS_ROLE_ARN=`aws cloudformation describe-stacks --stack-name ${STACK_NAME} --output json | jq -r '.Stacks[].Outputs[] | select(.OutputKey=="EKSAccessRoleARN") | .OutputValue'`
export AWS_ECR_NVME=`aws cloudformation describe-stacks --stack-name ${STACK_NAME} --output json | jq -r '.Stacks[].Outputs[] | select(.OutputKey=="EksNvmeProvisionerRepository") | .OutputValue'`
#export AWS_LOAD_GENERATOR = `aws cloudformation describe-stacks --stack-name ${STACK_NAME} --output json | jq -r '.Stacks[].Outputs[] | select(.OutputKey=="EksNvmeProvisionerRepository") | .OutputValue'`
export DYNAMODB_TABLENAME_PREFIX=${STACK_NAME}_

# test helm version release 3.8.2 is needed for eks 1.21 k8s
helm version

echo "[Setup] Granting access to the EKS cluster..."
make eks@grant-access EKS_ACCESS_ROLE_ARN=${EKS_ACCESS_ROLE_ARN} EKS_WORKER_ROLE_ARN=${EKS_WORKER_ROLE_ARN}

echo "[Setup] Login to the ECR registries..."
make ecr@login

make eks@cleanup
make benchmark@cleanup

echo "[Setup] Building the basic service on ARM64 and pushing it to the ECR registry..."
make eks@provision-codekit-`echo ${VARIANT,,}`

echo "[Setup] Building the bidder on ARM64 and pushing it to the ECR registry..."
 make bidder@build IMAGE_PREFIX="${STACK_NAME}-"
 make bidder@push IMAGE_PREFIX="${STACK_NAME}-"

echo "[Setup] Building the model on ARM64 and pushing it to the ECR registry..."
#make model@build IMAGE_PREFIX="${STACK_NAME}-"
#make model@push IMAGE_PREFIX="${STACK_NAME}-"

echo "[Setup] Building the nvme-provisioner and pushing it to the ECR registry..."
make buildx@install
make nvme-provisioner@build
#make nvme-provisioner@push

if sh -c "echo $VARIANT | grep -q -E '^(Aerospike)$'" ; then
    echo "[Setup] Deploying the Aerospike cluster"
    make eks@provision-nvme
    make aerospike@deploy AEROSPIKE_VARIANT="benchmark"
    make aerospike@wait
fi

if sh -c "echo $USE_DATAGEN | grep -q -E '^(yes)$'" ; then
    echo "[Setup] Populating the database with testing data..."
    make datagen@image IMAGE_PREFIX="${STACK_NAME}-"
    # make datagen@push IMAGE_PREFIX="${STACK_NAME}-"
    if sh -c "echo $VARIANT | grep -q -E '^(Aerospike)$'" ; then
       echo "Datagen on Aerospike has been disabled"
       make aerospike@datagen DATAGEN_CONCURRENCY=32 DATAGEN_ITEMS_PER_JOB=10000  DATAGEN_DEVICES_ITEMS_PER_JOB=100000 DATAGEN_DEVICES_PARALLELISM=30 STACK_NAME=${STACK_NAME}
    else
      make dynamodb@datagen DATAGEN_CONCURRENCY=1 DATAGEN_ITEMS_PER_JOB=1000  DATAGEN_DEVICES_ITEMS_PER_JOB=1000 DATAGEN_DEVICES_PARALLELISM=1 STACK_NAME=${STACK_NAME}
    fi
fi

# Deploy the bidderapp
export BIDDER_OVERLAY_TEMP=$(mktemp)
envsubst < deployment/infrastructure/deployment/bidder/overlay-codekit-${VARIANT,,}.yaml.tmpl >${BIDDER_OVERLAY_TEMP}
make eks@deploybidder VALUES=${BIDDER_OVERLAY_TEMP}

if sh -c "echo $USE_LOAD_GENERATOR | grep -q -E '^(yes)$'" ; then
    make load-generator@build
    # make load-generator@push
    LOAD_GENERATOR_OVERLAY_TEMP=$(mktemp)
    envsubst < deployment/infrastructure/deployment/load-generator/overlay-codekit.yaml.tmpl >${LOAD_GENERATOR_OVERLAY_TEMP}

    echo "[Setup] Deploying the load generator..."

    if sh -c "echo $VARIANT | grep -q -E '^(Aerospike)$'" ; then
      make benchmark@run TARGET="http://bidder/bidrequest" VALUES="${LOAD_GENERATOR_OVERLAY_TEMP}" RATE_PER_JOB=3000000 NUMBER_OF_JOBS=1 NUMBER_OF_DEVICES=1000000000 DURATION=30m STACK_NAME=${STACK_NAME} LOAD_GENERATOR_NODE_SELECTOR_POOL=benchmark
    else
      TARGET="http://bidder/bidrequest" VALUES="${LOAD_GENERATOR_OVERLAY_TEMP}" make benchmark@run-simple
    fi
fi
echo "[Setup] The bidder has been deployed. You can log in to the EKS cluster and access the Grafana dashboards."
