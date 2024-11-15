#!/usr/bin/env bash
# This script follows the steps outlined in CDP to setup the cloud 9 instance to run bench marks and 
# view results in graphana dashboard
# Login to cloud9 terminal, 
# cd RTBCodeKitRepo
# chmod 700 run-benchmark.sh
# ./run-benchmark.sh

set -ex
# take input parameters for the benchmark
if [ -z "$1" ]
then
    echo "Benchmark tool requires the cloud formation root stack name of the RTB Kit solution and the CLI profile"
    echo "Usage: ${0} <root-stack-name> <cli-profile> [<timeout>(100ms) <no:of jobs>(1) <rate per job>(200) <no:of devices>(1000) <duration>(60s)]"
    exit 1
fi
if [ -z "$2" ]
then
    echo "WARNING:AWS CLI Profile not given"
    export PROFILE="rtb"
    echo "Using default AWS CLI profile ${PROFILE}"
else
    export PROFILE=$2
fi
if [ -z "$3" ]
then
    echo "Timeout argument not given"
    export TIMEOUT=100ms
    echo "Using default timeout of ${TIMEOUT}"
fi
if [ -z "$4" ]
then
    echo "Number of jobs argument not given"
    export NUMBER_OF_JOBS=1
    echo "Using default number of jobs of ${NUMBER_OF_JOBS}"
fi
if [ -z "$5" ]
then
    echo "Rate per job argument not given"
    export RATE_PER_JOB=200
    echo "Using default rate per job of ${RATE_PER_JOB}"
fi
if [ -z "$6" ]
then
    echo "Number of devices argument not given"
    export NUMBER_OF_DEVICES=10000
    echo "Using default number of devices of ${NUMBER_OF_DEVICES}"
fi
if [ -z "$7" ]
then
    echo "Duration argument not given"
    export DURATION=60s
    echo "Using default duration of ${DURATION}"
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

# run benchmark
make benchmark@cleanup
make benchmark@run TIMEOUT=$TIMEOUT NUMBER_OF_JOBS=$NUMBER_OF_JOBS RATE_PER_JOB=$RATE_PER_JOB NUMBER_OF_DEVICES=$NUMBER_OF_DEVICES DURATION=$DURATION
# TIMEOUT=100ms        # Request timeout (default 100ms)
# DURATION=500s        # duration of the load generation
# RATE_PER_JOB=5000    # target request rate for the load generator
# NUMBER_OF_DEVICES=10 # number of device IFAs to use in bid request
# NUMBER_OF_JOBS=1     # number of parallel instances of the load generator
# SLOPE=0              # slope of requests per second increase (zero for a constant rate; see <https://en.wikipedia.org/wiki/Slope>)
# ENABLE_PROFILER=1    # used to start profiling session, leave unset to disable

# open graphana
kubectl port-forward svc/prom-grafana 8080:80
