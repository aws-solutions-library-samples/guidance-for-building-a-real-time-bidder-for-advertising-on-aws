#!/usr/bin/env bash
set -e

if [ "$#" -ne 3 ]; then
    echo "Usage: $0 app_name snapshot_name"
    exit 1
fi

app_name="$1"
name="$2"

dir=$(dirname "${BASH_SOURCE[0]}")

snapshotIds=$(aws ec2 describe-snapshots --filter "Name=tag:Name,Values=$name" --filter "Name=tag:App,Values=$app_name" --query "Snapshots[*].SnapshotId" --output text | tr "\t" "\n")
snapshotsCount=$(echo "$snapshotIds" | wc -l)

if [ $snapshotsCount -eq 0 ]; then
    echo "Snapshot '$name' not found"
    exit 1
fi

echo "Removing ${snapshotsCount} snapshots of size ${snapshotSize}Gi..."

index=0

for snapshotId in $snapshotIds; do
    aws ec2 delete-snapshot --snapshot-id $snapshotId
done

echo "Snapshots removed"
