# This script follows the steps outlined in CDP to setup the cloud 9 instance to run bench marks and monitor cluster
# Login to cloud9 terminal, 
# cd RTBCodeKitRepo
# chmod700 cloud9-setup.sh
# ./cloud9-setup.sh

#!/usr/bin/env bash
set -ex

# install helm specific version 3.8.2
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod 700 get_helm.sh && ./get_helm.sh --version v3.8.2

# install jq
sudo yum install jq

# install kubectl specific version 1.21.0
#curl -LO "https://dl.k8s.io/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl.sha256"
curl -LO "https://dl.k8s.io/release/v1.21.0/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

kubectl version --client --output=yaml    

# EKS Cluster connectivity
export AWS_ACCOUNT=""
export AWS_REGION="us-west-2"
export ROOT_STACK="rtbkit-bkr-dev"
export APPLICATION_STACK_NAME=`aws cloudformation list-exports --query "Exports[?Name=='ApplicationStackName'].Value" --output text`
export CODEBUILD_STACK_NAME=`aws cloudformation describe-stacks --stack-name ${ROOT_STACK} --output json | jq '.Stacks[].Outputs[] | select(.OutputKey=="CodebuildStackARN") | .OutputValue' | cut -d/ -f2`
export EKS_WORKER_ROLE_ARN=`aws cloudformation list-exports --query "Exports[?Name=='EKSWorkerRoleARN'].Value" --output text`
export EKS_ACCESS_ROLE_ARN=`aws cloudformation list-exports --query "Exports[?Name=='EKSAccessRoleARN'].Value" --output text`
export STACK_NAME=$APPLICATION_STACK_NAME

echo "You might need to update the aws cli configuration to use an existing permanent role if the Cloud9 temperory credentials dont work"

CREDS_JSON=`aws sts assume-role --role-arn $EKS_ACCESS_ROLE_ARN --role-session-name EKSRole-Session --output json`
export AWS_ACCESS_KEY_ID=`echo $CREDS_JSON | jq '.Credentials.AccessKeyId' | tr -d '"'`
export AWS_SECRET_ACCESS_KEY=`echo $CREDS_JSON | jq '.Credentials.SecretAccessKey' | tr -d '"'`
export AWS_SESSION_TOKEN=`echo $CREDS_JSON | jq '.Credentials.SessionToken' | tr -d '"'`
CREDS_JSON=""
make eks@grant-access
unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN

# connect to cluster
make eks@use
kubectl get pods

# run benchmark
make benchmark@cleanup
make benchmark@run TIMEOUT=100ms NUMBER_OF_JOBS=1 RATE_PER_JOB=200 NUMBER_OF_DEVICES=10000 DURATION=60s
# TIMEOUT=100ms        # Request timeout (default 100ms)
# DURATION=500s        # duration of the load generation
# RATE_PER_JOB=5000    # target request rate for the load generator
# NUMBER_OF_DEVICES=10 # number of device IFAs to use in bid request
# NUMBER_OF_JOBS=1     # number of parallel instances of the load generator
# SLOPE=0              # slope of requests per second increase (zero for a constant rate; see <https://en.wikipedia.org/wiki/Slope>)
# ENABLE_PROFILER=1    # used to start profiling session, leave unset to disable

# open graphana
kubectl port-forward svc/prom-grafana 8080:80
