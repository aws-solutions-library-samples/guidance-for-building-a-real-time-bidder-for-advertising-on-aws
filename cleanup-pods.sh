#!/usr/bin/env bash
# This script is a helper to check logs from the pods

set -e
# take input parameters for the benchmark
kubectl get pods

pods=`kubectl get pods | grep -v "NAME" | sed -r 's/[ ]+/|/g'| cut -f 1,3 -d "|"`

if [ -z "$pods" ]
then
    echo "Pod Not found!"
    exit 0
fi

while IFS= read -r line; do
    echo "$line"
    pod=`echo $line | cut -f 1 -d "|"`
    status=`echo $line | cut -f 2 -d "|"`
    echo "$pod"
    echo "$status"
    if [ "$status" != "Running" ]
    then
        echo "deleting pod ${pod}"
        kubectl delete pod $pod
    fi
done <<< "$pods"

kubectl get pods
