#!/usr/bin/env bash
# This script is a helper to get logs from the eks pods

set -e
# take input parameters for the benchmark
if [ -z "$1" ]
then
    echo "Benchmark tool requires the cloud formation root stack name of the RTB Kit solution and the CLI profile"
    echo "Usage: ${0} <root-stack-name> <pod prefix> [<profile> <log option[head|tail]>]"
    exit 1
fi
if [ -z "$2" ]
then
    echo "Enter the pod prefix"
    echo "Usage: ${0} <root-stack-name> <pod prefix> [<profile> <log option[head|tail]>]"
    echo "Example: ${0} load-generator"
    exit 1
fi
if [ -z "$3" ]
then
    echo "WARNING:AWS CLI Profile not given"
    export PROFILE="rtb"
    echo "Using default AWS CLI profile ${PROFILE}"
else
    export PROFILE=$2
fi
if [ -z "$4" ]
then
    echo "Log option not given"
fi

# EKS Cluster connectivity
export AWS_ACCOUNT=$(aws sts get-caller-identity --query Account --output text --profile $PROFILE)
export AWS_REGION=$(aws configure get region --profile $PROFILE)
export ROOT_STACK=$1
export APPLICATION_STACK_NAME=`aws cloudformation list-exports --query "Exports[?Name=='ApplicationStackName'].Value" --output text --profile ${PROFILE}`
export CODEBUILD_STACK_NAME=`aws cloudformation describe-stacks --stack-name ${ROOT_STACK} --output json --profile ${PROFILE} | jq '.Stacks[].Outputs[] | select(.OutputKey=="CodebuildStackARN") | .OutputValue' | cut -d/ -f2`
export EKS_WORKER_ROLE_ARN=`aws cloudformation list-exports --query "Exports[?Name=='EKSWorkerRoleARN'].Value" --output text --profile ${PROFILE}`
export EKS_ACCESS_ROLE_ARN=`aws cloudformation list-exports --query "Exports[?Name=='EKSAccessRoleARN'].Value" --output text --profile ${PROFILE}`
export STACK_NAME=$ROOT_STACK

echo "Check the CLI profile configuration, account and region settings first if you hit errors"

CREDS_JSON=`aws sts assume-role --role-arn $EKS_ACCESS_ROLE_ARN --role-session-name EKSRole-Session --output json --profile $PROFILE`
export AWS_ACCESS_KEY_ID=`echo $CREDS_JSON | jq '.Credentials.AccessKeyId' | tr -d '"'`
export AWS_SECRET_ACCESS_KEY=`echo $CREDS_JSON | jq '.Credentials.SecretAccessKey' | tr -d '"'`
export AWS_SESSION_TOKEN=`echo $CREDS_JSON | jq '.Credentials.SessionToken' | tr -d '"'`
CREDS_JSON=""
make eks@grant-access

# connect to cluster
# make eks@use
kubectl get pods

pods=`kubectl get pods | grep "${2}" |sed -r 's/[ ]+/|/g'| cut -f 1,3 -d "|"`

if [ -z "$pods" ]
then
    echo "Pod Not found!"
    exit 0
fi

while IFS= read -r line; do
    pod=`echo $line | cut -f 1 -d "|"`
    echo "$pod"
    if [ "$2" = "head" ]
    then
        echo "Getting head"
        
        kubectl logs $pod | head
    elif [ "$2" = "tail" ]
    then
        echo "Getting tail"
        kubectl logs $pod | tail
    
    else
        echo "Getting all logs"
        kubectl logs $pod
    fi
done <<< "$pods"
