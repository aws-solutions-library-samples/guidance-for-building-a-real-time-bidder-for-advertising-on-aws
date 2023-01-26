snapshot@create: ## Create EBS snapshot of PVC volumes
	./tools/snapshots/snapshot-create.sh \
		$(SNAPSHOT_APP) \
		$(SNAPSHOT_PVC_PREFIX) \
		$(if $(SNAPSHOT_NAME),$(SNAPSHOT_NAME),$(shell date +$(STACK_NAME)-%Y%m%d%H%M))

snapshot@restore: ## Restores PVCs from EBS snapshot
	./tools/snapshots/snapshot-restore.sh $(SNAPSHOT_APP) $(SNAPSHOT_PVC_PREFIX) $(SNAPSHOT_NAME)

snapshot@delete: ## Removes EBS snapshot
	./tools/snapshots/snapshot-delete.sh $(SNAPSHOT_APP) $(SNAPSHOT_NAME)
