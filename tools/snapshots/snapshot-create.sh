#!/usr/bin/env bash
set -e

if [ "$#" -ne 3 ]; then
    echo "Usage: $0 app_name pvc_prefix snapshot_name"
    exit 1
fi

app_name="$1"
pvc_prefix="$2"
name="$3"

dir=$(dirname "${BASH_SOURCE[0]}")
pvcs=$(kubectl get pvc -o custom-columns=NAME:.metadata.name --no-headers | grep $pvc_prefix)

function check_tag {
    output=$(aws ec2 describe-snapshots --filter "Name=tag:Name,Values=$app_name-$name" --output text)

    if [ -n "$output" ]; then
      echo "Snapshot '$name' for application '$app_name' already exists. Use different name or remove previous snapshot."
      exit 1
    fi
}

function create_snapshots {
    echo "Creating snapshots..."

    for pvc in $pvcs; do
        true
        PVC_NAME=$pvc envsubst < "$dir/templates/snapshot.yaml" | kubectl apply -f -
    done;
}

function wait_for_snapshots {
    echo -n "Waiting for snapshots to be ready..."

    while true; do
        for pvc in $pvcs; do
            status=$(kubectl get volumesnapshots $pvc -o jsonpath='{.status.readyToUse}')

            if [ "$status" = "false" ]; then
                echo -n "."
                sleep 2
                continue 2;
            fi;
        done;

        echo
        break;
    done;
}

function tag_snapshots {
    echo "Tagging snapshots on AWS..."

    for pvc in $pvcs; do
        snapshotContentName=$(kubectl get volumesnapshots $pvc -o jsonpath='{.status.boundVolumeSnapshotContentName}')
        snapshotId=$(kubectl get volumesnapshotcontents $snapshotContentName -o jsonpath='{.status.snapshotHandle}')

        aws ec2 create-tags --resources $snapshotId --tags "Key=Name,Value=$app_name-$name" "Key=PVC,Value=$pvc"
    done;
}

function remove_k8s_snapshots {
    echo "Removing snapshots from K8s..."

    for pvc in $pvcs; do
        snapshotContentName=$(kubectl get volumesnapshots $pvc -o jsonpath='{.status.boundVolumeSnapshotContentName}')

        kubectl delete volumesnapshots/$pvc
        kubectl delete volumesnapshotcontents/$snapshotContentName
    done
}

check_tag
create_snapshots
wait_for_snapshots
tag_snapshots
remove_k8s_snapshots

echo "Snapshots '$name' created."
