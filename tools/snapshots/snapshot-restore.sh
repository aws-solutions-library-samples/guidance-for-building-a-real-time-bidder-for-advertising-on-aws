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

snapshotIds=$(aws ec2 describe-snapshots --filter "Name=tag:Name,Values=$app_name-$name" --query "Snapshots[*].SnapshotId" --output text | tr "\t" "\n")
snapshotsCount=$(echo "$snapshotIds" | wc -l)

if [ $snapshotsCount -eq 0 ]; then
    echo "Snapshot '$name' not found"
    exit 1
fi

snapshotSize=$(aws ec2 describe-snapshots --snapshot-ids $snapshotIds --query 'Snapshots[0].VolumeSize' --output text)

echo "Import ${snapshotsCount} snapshots of size ${snapshotSize}Gi..."

index=0

for snapshotId in $snapshotIds; do
    name="$pvc_prefix-$index"

    export APP_NAME=$app_name
    export SNAPSHOT_NAME=$name
    export CONTENT_NAME=$name
    export PVC_NAME=$name
    export SNAPSHOT_ID=$snapshotId
    export SNAPSHOT_SIZE=$snapshotSize

    envsubst < "$dir/templates/import-snapshot.yaml" | kubectl apply -f -
    envsubst < "$dir/templates/import-snapshot-content.yaml"  | kubectl apply -f -
    envsubst < "$dir/templates/import-pvc.yaml"  | kubectl apply -f -

    index=$((index + 1))
done

echo "Snapshot restored"
