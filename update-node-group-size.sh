#!/usr/bin/env bash
set -e

# take the desired node size as input
if [ -z "$1" ]
then
    echo "This script sets node size of all the node groups in the Script requires the cloud formation root stack name of the RTB Kit solution"
    echo "Usage: ${0} <cluster-name> <desired node size>"
    exit 1
fi
if [ -z "$2" ]
then
    echo "Script requires the desired size of node group"
    echo "Usage: ${0} <cluster-name> <desired node size>"
    exit 1
fi
clustername=$1
size=$2
nodegroups=`aws eks list-nodegroups --cluster-name $clustername | jq '.nodegroups[]' | tr -d "\""`

while IFS= read -r line; do
    echo $line
    # if you want to change the min size as well
    # aws eks update-nodegroup-config \
    #   --cluster-name $clustername \
    #   --nodegroup-name $line \
    #   --scaling-config minSize=$size
    aws eks update-nodegroup-config \
      --cluster-name $clustername \
      --nodegroup-name $line \
      --scaling-config desiredSize=$size
done <<< "$nodegroups"
