#!/bin/bash
# take region and cluster name and cli profile as input variables
# take input parameters for the benchmark
if [ -z "$1" ]
then
    echo "This script requires the aws region cloud formation root stack name of the RTB Kit solution"
    echo "Usage: ${0} <aws-region> <root-stack-name>"
    exit 1
fi
if [ -z "$2" ]
then
    echo "This script requires the aws region cloud formation root stack name of the RTB Kit solution"
    echo "Usage: ${0} <aws-region> <root-stack-name>"
    exit 1
fi

AWS_REGION=$1
RTBKIT_ROOT_STACK_NAME=$2
LoggingConfig='{"clusterLogging":[{"types":["api","audit","authenticator","controllerManager","scheduler"],"enabled":true}]}'
echo "Starting cluster config update."

if [ -z "$3" ]
then
    echo "WARNING:AWS CLI Profile not given"
    output=`aws eks update-cluster-config --region $AWS_REGION --name $RTBKIT_ROOT_STACK_NAME --logging $LoggingConfig 2>&1`

else
    export PROFILE=$3
    output=`aws eks update-cluster-config --region $AWS_REGION --name $RTBKIT_ROOT_STACK_NAME --logging $LoggingConfig --profile $PROFILE 2>&1`  
fi
# if output contains "No changes needed for the logging config provided", then the cluster config was already updated
# otherwise, there was an error
if [[ $output == *"No changes needed for the logging config provided"* ]]; then
    echo "Cluster config already updated."
    exit 0
else
    echo "Error encountered while updating the cluster logging config"
    echo "ERROR:${output}"
    exit 2
fi
echo "Cluster config updated successfully."