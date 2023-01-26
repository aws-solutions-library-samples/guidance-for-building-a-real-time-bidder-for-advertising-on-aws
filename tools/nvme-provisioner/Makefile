include $(dir $(lastword $(MAKEFILE_LIST)))/../../config.mk

NVME_PROVISIONER_IMAGE = $(AWS_PUBLIC_ECR_REGISTRY)/$(AWS_PUBLIC_IMAGE_PREFIX)nvme-provisioner
NVME_PROVISIONER_TAG = latest

nvme-provisioner@build:
	cd tools/nvme-provisioner; \
	docker buildx build --push --platform linux/amd64,linux/arm64 -t $(NVME_PROVISIONER_IMAGE):$(NVME_PROVISIONER_TAG) .

nvme-provisioner@push:
	docker push $(NVME_PROVISIONER_IMAGE):$(NVME_PROVISIONER_TAG)
