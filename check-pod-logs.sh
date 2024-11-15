#!/usr/bin/env bash
# This script is a helper to cleanup unhealthy pods

set -e
# take input parameters for the benchmark
if [ -z "$1" ]
then
    echo "Enter the pod prefix"
    echo "Usage: ${0} <pod prefix> [head|tail]"
    echo "Example: ${0} load-generator"
    exit 1
fi
if [ -z "$2" ]
then
    echo "Log option not given"
fi

kubectl get pods

pods=`kubectl get pods | grep "${1}" |sed -r 's/[ ]+/|/g'| cut -f 1,3 -d "|"`

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
