_PWD := $(shell pwd | sed "s~/cygdrive/c~c:~")
COMMA := ,
CF_BUCKET_REGION=us-west-2
STACK_NAME ?= bidder
DYNAMODB_TEMPLATE_FILE ?= dynamodb.yaml
DYNAMODB_STACK_NAME ?= dynamodb
TEMPLATE_BUCKET := aws-bidder-cf-templates
TEMP_FILE := $(shell mktemp)
EKS_ACCESS_ROLE_ARN ?= $(shell aws cloudformation list-exports --query "Exports[?Name=='EKSAccessRoleARN'].Value" --output text)
EKS_WORKER_ROLE_ARN ?= $(shell aws cloudformation list-exports --query "Exports[?Name=='EKSWorkerRoleARN'].Value" --output text)
AEROSPIKE_NODE_COUNT = $(shell kubectl get nodes -l pool=aerospike --no-headers | wc -l  | tr -d ' ')
AWS_USERNAME = $(shell aws sts get-caller-identity --output json | jq -r  '.Arn' | cut -d/ -f2)
AWS_REGION ?= us-west-2
AWS_TAGS = "StackName=$(STACK_NAME)" "Variant=$(VARIANT)" "Owner=$(AWS_USERNAME)"
K8S_AWS_TAGS = "StackName=$(STACK_NAME),Variant=$(VARIANT),Owner=$(AWS_USERNAME)"


IMAGE_PREFIX=$(STACK_NAME)-

CFN_NAG_TAG ?= stelligent/cfn_nag:0.7.12

CFN_NAG_RUN = docker run -v "$(_PWD):/templates" --entrypoint cfn_nag_scan $(CFN_NAG_TAG)

# Variant of the infrastructure configuration (see: application.yaml)
VARIANT ?= Basic
# If not basic, apply bidder deployment values from deployment/bidder/overlay-$(OVERLAY).yaml.
OVERLAY ?= basic
# Parameters passed to all Helm invocations on our charts.
BASE_HELM_OPTS = --set image.registry=$(AWS_ECR_REGISTRY)
# Additional parameters to helm install or helm upgrade. Use e.g. HELM_OPTS=--dry-run to see what objects would be
# deployed.
HELM_OPTS ?=
# If not empty, use as a path a custom values.yaml (not using ones from OVERLAY). Pass different files to commands
# deploying the bidder or the load generator.
VALUES ?=


# Image versions: override if testing new features available on a tag.
# BIDDER_IMAGE_VERSION ?= latest
LOAD_GENERATOR_IMAGE_VERSION ?= latest

# Run load generator on benchmark nodes if using a benchmark variant, otherwise on basic-arm nodes.
#LOAD_GENERATOR_NODE_SELECTOR_POOL ?= $(if $(findstring Benchmark,$(VARIANT)),benchmark,basic-arm)
LOAD_GENERATOR_NODE_SELECTOR_POOL ?= benchmark
# Benchmark profiler output variables
NAME ?= $(VARIANT)
DATE ?= $(shell date +%Y-%m-%dT%H:%M)
PROFILER_OUTPUT ?= $(DATE)-$(NAME)/pprof-{{.Endpoint}}-{{.Hostname}}

# Aerospike variant to deploy. If not basic, apply additional
# configuration from deployment/infrastructure/deployment/aerospike/$(AEROSPIKE_VARIANT)-*
AEROSPIKE_VARIANT ?= basic

ifeq ("$(wildcard deployment/infrastructure/ci/$(PROJECT_NAME)-stack\.yaml)","")
	PROJECT_NAME_IS_VALID=$(error PROJECT_NAME is invalid, look at the *-stack.yaml files in deployment/infrastructure/ci)
endif


# Validate the parameters required by ci@* targets
_validate@ci:
	$(PROJECT_NAME_IS_VALID)

stack@package: ## Process and upload CloudFormation templates to S3
	AWS_REGION=$(CF_BUCKET_REGION) aws cloudformation package \
		--template-file $(_PWD)/deployment/infrastructure/application.yaml \
		--output-template-file $(TEMP_FILE) \
		--s3-bucket $(TEMPLATE_BUCKET) \
		--s3-prefix app


stack@deploy:
stack@deploy: stack@package ## Deploy infrastructure. Specify which `VARIANT` to deploy, eg `VARIANT=Basic`
	aws cloudformation deploy --stack-name $(STACK_NAME) \
		--template-file $(TEMP_FILE) \
		--capabilities CAPABILITY_IAM CAPABILITY_AUTO_EXPAND \
		--parameter-overrides "Variant=$(VARIANT)" \
		--tags $(AWS_TAGS)

stack@delete: ## Delete CloudFormation stack with all infrastructure
	@read -p "Type '$(STACK_NAME)' to confirm stack delete: " name && ([ "$${name}" == "$(STACK_NAME)" ] || exit 1)
	aws cloudformation delete-stack --stack-name $(STACK_NAME)
	@aws cloudformation wait stack-delete-complete --stack-name $(STACK_NAME) || (\
		aws cloudformation describe-stack-resources --stack-name $(STACK_NAME) \
		--query "StackResources[?ResourceStatus=='DELETE_FAILED']" \
		exit 1 \
	)

ci@deploy: _validate@ci
ci@deploy: ## Deploy the Codebuild project on AWS.
	aws cloudformation deploy --stack-name $(STACK_NAME) \
		--template-file deployment/infrastructure/ci/$(STACK_NAME)-stack.yaml \
		--capabilities CAPABILITY_IAM CAPABILITY_AUTO_EXPAND CAPABILITY_NAMED_IAM

ci@delete: _validate@ci
ci@delete: ## Delete CloudFormation stack with the given CodeBuild project.
	@read -p "Type '$(STACK_NAME)' to confirm stack delete: " name && ([ "$${name}" == "$(STACK_NAME)" ] || exit 1)
	aws cloudformation delete-stack --stack-name $(STACK_NAME)
	@aws cloudformation wait stack-delete-complete --stack-name $(STACK_NAME) || (\
		aws cloudformation describe-stack-resources --stack-name $(STACK_NAME) \
		--query "StackResources[?ResourceStatus=='DELETE_FAILED']" \
		exit 1 \
	)

eks@use: ## Configure kubectl config to EKS cluster created for specified stack
	aws eks update-kubeconfig --name "$(STACK_NAME)" --role-arn $(EKS_ACCESS_ROLE_ARN)

eks@grant-access: ## Grant access to EKS cluster for rest of the team
	aws eks update-kubeconfig --name "$(STACK_NAME)"
	EKS_ACCESS_ROLE_ARN=$(EKS_ACCESS_ROLE_ARN) EKS_WORKER_ROLE_ARN=$(EKS_WORKER_ROLE_ARN) \
	envsubst < deployment/infrastructure/deployment/aws-auth.tpl.yaml > $(TEMP_FILE)
	kubectl apply -f $(TEMP_FILE)

eks@provision: eks@update-dashboards ## Provision basic services to EKS cluster
eks@provision: eks@provision-external-dns
eks@provision: eks@provision-prometheus
eks@provision: eks@provision-prometheus-adapter
eks@provision: eks@provision-autoscaler
eks@provision: eks@provision-ebs-csi
eks@provision: eks@provision-nvme

eks@provision-codekit-aerospike: eks@update-dashboards ## Provision basic services to EKS cluster
eks@provision-codekit-aerospike: eks@provision-prometheus
eks@provision-codekit-aerospike: eks@provision-prometheus-adapter
eks@provision-codekit-aerospike: eks@provision-ebs-csi
eks@provision-codekit-aerospike: eks@provision-nvme
eks@provision-codekit-aerospike: eks@provision-autoscaler

eks@provision-codekit-dynamodb: eks@update-dashboards ## Provision basic services to EKS cluster
eks@provision-codekit-dynamodb: eks@provision-prometheus
eks@provision-codekit-dynamodb: eks@provision-prometheus-adapter
eks@provision-codekit-dynamodb: eks@provision-ebs-csi
eks@provision-codekit-dynamodb: eks@provision-nvme
eks@provision-codekit-dynamodb: eks@provision-autoscaler

eks@provision-codekit-dynamodbbasic: eks@update-dashboards ## Provision basic services to EKS cluster
eks@provision-codekit-dynamodbbasic: eks@provision-prometheus
eks@provision-codekit-dynamodbbasic: eks@provision-prometheus-adapter
eks@provision-codekit-dynamodbbasic: eks@provision-ebs-csi
eks@provision-codekit-dynamodbbasic: eks@provision-nvme
eks@provision-codekit-dynamodbbasic: eks@provision-autoscaler

eks@update-dashboards: ## Update Grafana dashboards on EKS
	kubectl apply -k deployment/infrastructure/deployment/prometheus/dashboards

eks@provision-external-dns: ## Provision External DNS
	kubectl apply -f deployment/infrastructure/deployment/external-dns.yaml

eks@provision-prometheus: ## Provision Prometheus stack, adding a timeout parameter for resiliency
	helm repo add --force-update prometheus-community https://prometheus-community.github.io/helm-charts
	helm upgrade --install \
		--version 59.0 \
		prom prometheus-community/kube-prometheus-stack \
		-f deployment/infrastructure/deployment/prometheus/prometheus-values.yaml \
		--timeout 10m30s --debug

eks@provision-prometheus-adapter: ## Provision Prometheus adapter for custom metrics API
	helm repo add --force-update prometheus-community https://prometheus-community.github.io/helm-charts
	helm upgrade --install \
		--version 4.2.0 \
		prom-adapter prometheus-community/prometheus-adapter \
		-f deployment/infrastructure/deployment/prometheus/adapter-values.yaml

eks@provision-autoscaler: ## Provision Cluster Autoscaler
	helm repo add --force-update autoscaler https://kubernetes.github.io/autoscaler
	helm upgrade --install \
		--version 9.37.0 \
		autoscaler autoscaler/cluster-autoscaler \
		-f deployment/infrastructure/deployment/autoscaler/values.yaml \
		--set "autoDiscovery.clusterName=$(STACK_NAME)"

eks@provision-aws-loadbalancer-controller: ## Provision Cluster Autoscaler
	helm repo add --force-update eks https://aws.github.io/eks-charts
	helm upgrade --install aws-load-balancer-controller eks/aws-load-balancer-controller \
  		-n kube-system \
  		--set clusterName=$(STACK_NAME) \
  		--set serviceAccount.create=true 

eks@deploy-loadbalancers:
# deploy external NLB to test bidder through public internet
	kubectl apply -f deployment/infrastructure/deployment/bidder-external.yaml
# deploy external NLB to access Grafana
	kubectl apply -f deployment/infrastructure/deployment/grafana-external.yaml


eks@provision-ebs-csi: ## Provision EBS CSI driver
	helm repo add --force-update aws-ebs-csi-driver https://kubernetes-sigs.github.io/aws-ebs-csi-driver
	helm upgrade --install \
		--version 2.30.0 \
		aws-ebs-csi-driver aws-ebs-csi-driver/aws-ebs-csi-driver \
		-f deployment/infrastructure/deployment/ebs-csi-driver/values.yaml
	kubectl apply -f deployment/infrastructure/deployment/ebs-csi-driver/crd
	kubectl apply -f deployment/infrastructure/deployment/ebs-csi-driver/resources

eks@provision-nvme: export IMAGE_PREFIX=$(STACK_NAME)-
eks@provision-nvme: export IMAGE_TAG=latest
eks@provision-nvme: ## Provision NVMe volume provisioner
	envsubst < deployment/infrastructure/deployment/nvme-provisioner.yaml.tmpl | kubectl apply -f -


eks@deploybidder: BIDDER_IMAGE_REPOSITORY="$(STACK_NAME)-bidder"
eks@deploybidder: ## Deploy application to EKS
# Pass a values overlay file if not using the basic overlay.
	helm upgrade --install $(BASE_HELM_OPTS) $(HELM_OPTS) bidder ./deployment/infrastructure/charts/bidder \
		$(foreach ITEM,$(subst $(COMMA), ,$(OVERLAY)), $(if $(strip $(subst basic,,$(ITEM))),-f ./deployment/infrastructure/deployment/bidder/overlay-$(ITEM).yaml,)) \
		$(and $(findstring Kinesis,$(VARIANT)),--set useBenchmarkKinesis=true) \
		$(and $(strip $(BIDDER_IMAGE_VERSION)),--set image.tag=$(BIDDER_IMAGE_VERSION)) \
		$(and $(strip $(BIDDER_IMAGE_REPOSITORY)),--set image.repository=$(BIDDER_IMAGE_REPOSITORY)) \
		$(if $(SERVICE_COUNT),--set service.count=$(SERVICE_COUNT),) \
		--set stackName=$(STACK_NAME) \
		--set awsRegion=$(AWS_REGION) \
		--set service.annotations."service\.beta\.kubernetes\.io/aws-load-balancer-additional-resource-tags"=$(K8S_AWS_TAGS) \
		$(if $(CERTIFICATE_UUID),--set service.annotations."service\.beta\.kubernetes\.io/aws-load-balancer-ssl-cert"=arn:aws:acm:$(AWS_REGION):$(AWS_ACCOUNT):certificate/$(CERTIFICATE_UUID),) \
		$(if $(VALUES),-f $(VALUES),)

eks@deploymodel: MODEL_IMAGE_REPOSITORY="$(STACK_NAME)-model"
eks@deploymodel:
	helm upgrade --install $(BASE_HELM_OPTS) $(HELM_OPTS) model ./deployment/infrastructure/charts/model \
		$(foreach ITEM,$(subst $(COMMA), ,$(OVERLAY)), $(if $(strip $(subst basic,,$(ITEM))),-f ./deployment/infrastructure/deployment/model/overlay-$(ITEM).yaml,)) \
		$(and $(strip $(MODEL_IMAGE_VERSION)),--set image.tag=$(MODEL_IMAGE_VERSION)) \
		$(and $(strip $(MODEL_IMAGE_REPOSITORY)),--set image.repository=$(MODEL_IMAGE_REPOSITORY)) \
		$(if $(SERVICE_COUNT),--set service.count=$(SERVICE_COUNT),) \
		--set stackName=$(STACK_NAME) \
		--set awsRegion=$(AWS_REGION) \
		--set service.annotations."service\.beta\.kubernetes\.io/aws-load-balancer-additional-resource-tags"=$(K8S_AWS_TAGS) \
        $(if $(CERTIFICATE_UUID),--set service.annotations."service\.beta\.kubernetes\.io/aws-load-balancer-ssl-cert"=arn:aws:acm:$(AWS_REGION):$(AWS_ACCOUNT):certificate/$(CERTIFICATE_UUID),) \
        $(if $(VALUES),-f $(VALUES),)

eks@cleanup: ## Remove application from EKS
	_=$(helm uninstall bidder 2>/dev/null)
	_=$(helm uninstall model 2>/dev/null)

publisher-eks@provision:
	export DESIRED_VERSION="v3.17.3"
	curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
	envsubst < deployment/infrastructure/publisher-eks/cluster.yaml | eksctl create cluster -f  -
	kubectl apply -f deployment/infrastructure/publisher-eks/nodepool.yaml

## All variables passed to the load generator are listed here with their defaults; remember to specify duration unit.
benchmark@run: LOAD_GENERATOR_OVERLAY_TEMP=$(shell mktemp)
benchmark@run: VALUES="$(LOAD_GENERATOR_OVERLAY_TEMP)"
benchmark@run: IMAGE_PREFIX=$(STACK_NAME)-
benchmark@run: ## Start benchmark
	envsubst < deployment/infrastructure/deployment/load-generator/overlay-codekit.yaml.tmpl > $(LOAD_GENERATOR_OVERLAY_TEMP) && \
	 helm upgrade --install $(BASE_HELM_OPTS) $(HELM_OPTS) load-generator ./deployment/infrastructure/charts/load-generator \
		$(and $(DURATION),--set duration=$(DURATION)) \
		$(and $(TIMEOUT),--set timeout=$(TIMEOUT)) \
		$(and $(START_DELAY),--set start-delay=$(START_DELAY)) \
		$(and $(NUMBER_OF_JOBS),--set numberOfJobs=$(NUMBER_OF_JOBS)) \
		$(and $(NUMBER_OF_DEVICES),--set devicesUsed=$(NUMBER_OF_DEVICES)) \
		$(and $(RATE_PER_JOB),--set ratePerJob=$(RATE_PER_JOB)) \
		$(and $(ENABLE_PROFILER),--set enableProfiler=true) \
		$(if $(TARGET),--set target=$(TARGET),) \
		$(if $(SERVICE_COUNT),--set targets.dynamic.count=$(SERVICE_COUNT),) \
		--set profilerOutput=$(PROFILER_OUTPUT) \
		--set image.tag=$(LOAD_GENERATOR_IMAGE_VERSION) \
		--set awsRegion=$(AWS_REGION) \
		--set nodeSelector.pool=$(LOAD_GENERATOR_NODE_SELECTOR_POOL) \
		--set stackName=$(STACK_NAME) \
		--set image.repository=$(IMAGE_PREFIX)load-generator \
		--set waitForService.image=$(AWS_PUBLIC_IMAGE_PREFIX)alpine \
		--set public_ecr_registry=$(AWS_PUBLIC_ECR_REGISTRY)
		--set service.annotations."service\.beta\.kubernetes\.io/aws-load-balancer-additional-resource-tags"=$(K8S_AWS_TAGS) \
		$(if $(VALUES),-f $(VALUES),)

benchmark@run-simple:
benchmark@run-simple: ## Start benchmark
	 helm upgrade --install $(BASE_HELM_OPTS) $(HELM_OPTS) load-generator ./deployment/infrastructure/charts/load-generator \
		--set image.tag=$(LOAD_GENERATOR_IMAGE_VERSION) \
		--set awsRegion=$(AWS_REGION) \
		--set target=$(TARGET)\
		--set stackName=$(STACK_NAME) \
		--set image.repository=$(IMAGE_PREFIX)load-generator \
		--set waitForService.image=$(AWS_PUBLIC_IMAGE_PREFIX)alpine \
		--set public_ecr_registry=$(AWS_PUBLIC_ECR_REGISTRY)
		--set service.annotations."service.beta\.kubernetes\.io/aws-load-balancer-additional-resource-tags"=$(K8S_AWS_TAGS) \
		$(if $(SERVICE_COUNT),--set targets.dynamic.count=$(SERVICE_COUNT),) \
		$(if $(VALUES),-f $(VALUES),)

benchmark@wait: ## Wait for load generator to complete
	kubectl wait --for=condition=Complete --timeout=-1s job/load-generator


benchmark@cleanup: ## Cleanup pods after benchmark
	_=$(helm uninstall load-generator)
#	helm uninstall report-aggregator

benchmark@cleanup-bidder: ## Cleanup pods after benchmark
	_=$(helm uninstall bidder)

LOAD_GENERATOR_OVERLAY_TEMP:=$(shell mktemp)
benchmark@codekit: ## Start a Codekit benchmark
		envsubst < deployment/infrastructure/deployment/load-generator/overlay-codekit.yaml.tmpl > $(LOAD_GENERATOR_OVERLAY_TEMP) && \
			  $(MAKE) benchmark@run IMAGE_PREFIX=$(IMAGE_PREFIX) \
				RATE_PER_JOB=0 NUMBER_OF_JOBS=1 NUMBER_OF_DEVICES=1000 DURATION=10m VALUES="$(LOAD_GENERATOR_OVERLAY_TEMP)"


BIDDER_OVERLAY_TEMP:=$(shell mktemp)
#MODEL_OVERLAY_TEMP:=$(shell mktemp)
bidder@deploy-codekit:
bidder@deploy-codekit: DYNAMODB_TABLENAME_PREFIX=$(STACK_NAME_)
bidder@deploy-codekit: VARIANT=dynamodb
bidder@deploy-codekit: IMAGE_PREFIX?=$(STACK_NAME)-
bidder@deploy-codekit: BIDDER_IMAGE_REPOSITORY="$(STACK_NAME)-bidder"
#bidder@deploy-codekit: MODEL_IMAGE_REPOSITORY="$(STACK_NAME)-model"
bidder@deploy-codekit: ## Start a Codekit benchmark
	envsubst < deployment/infrastructure/deployment/bidder/overlay-codekit-$(VARIANT).yaml.tmpl >$(BIDDER_OVERLAY_TEMP) && \
		$(MAKE) eks@deploybidder IMAGE_PREFIX=$(IMAGE_PREFIX) VALUES=$(BIDDER_OVERLAY_TEMP) \
		AWS_REGION=$(AWS_REGION) BIDDER_IMAGE_REPOSITORY="$(IMAGE_PREFIX)bidder" AWS_ACCOUNT=$(AWS_ACCOUNT)
#	envsubst < deployment/infrastructure/deployment/model/overlay-codekit.yaml.tmpl >$(MODEL_OVERLAY_TEMP) && \
        	$(MAKE) eks@deploymodel IMAGE_PREFIX=$(IMAGE_PREFIX) VALUES=$(MODEL_OVERLAY_TEMP) \
        	AWS_REGION=$(AWS_REGION) MODEL_IMAGE_REPOSITORY="$(IMAGE_PREFIX)model" AWS_ACCOUNT=$(AWS_ACCOUNT)

benchmark@report:
benchmark@report: ## Prepare aggregated report
	helm upgrade --install $(BASE_HELM_OPTS) $(HELM_OPTS) report-aggregator ./deployment/infrastructure/charts/report-aggregator \
		--set awsRegion=$(AWS_REGION) \
		--set image.tag=$(or $(REPORT_AGGREGATOR_IMAGE_VERSION),latest) \
		--set nodeSelector.pool=$(LOAD_GENERATOR_NODE_SELECTOR_POOL) \
		--set service.annotations."service\.beta\.kubernetes\.io/aws-load-balancer-additional-resource-tags"=$(K8S_AWS_TAGS) \
		$(if $(VALUES),-f $(VALUES),)
	kubectl wait --for=condition=Complete --timeout=-1s job/report-aggregator
# Docusaurus special formatting (first - is escaped)
	$(if $(REPORT_FILE),printf "\\x2d--\\nsidebar: false\\n---\\n\`\`\`\\n" > $(REPORT_FILE))
	kubectl logs job/report-aggregator $(if $(REPORT_FILE),>> $(REPORT_FILE),)
	$(if $(REPORT_FILE),echo "\`\`\`" >> $(REPORT_FILE),)

dynamodb@update: ## Scale DynamoDB for given VARIANT
	aws cloudformation package --template-file $(_PWD)/deployment/infrastructure/$(DYNAMODB_TEMPLATE_FILE) --output-template-file $(TEMP_FILE) --s3-bucket $(TEMPLATE_BUCKET) --s3-prefix dynamodb
	aws cloudformation deploy --stack-name $(DYNAMODB_STACK_NAME) \
			 --template-file $(TEMP_FILE) \
			 --capabilities CAPABILITY_IAM \
			 --parameter-overrides "Variant=$(VARIANT)" \
			 --tags $(AWS_TAGS)
	$(MAKE) dynamodb@check

dynamodb@check: ## Check DynamoDB stack for drifts
	@DRIFT_ID=$$(aws cloudformation detect-stack-drift --stack-name $(DYNAMODB_STACK_NAME) --query="StackDriftDetectionId" --output text); \
	until [ "$$(aws cloudformation describe-stack-drift-detection-status --stack-drift-detection-id $$DRIFT_ID --query "DetectionStatus" --output text)" == DETECTION_COMPLETE ]; do \
	  sleep 1; \
	done; \
	if [ "$$(aws cloudformation describe-stack-drift-detection-status --stack-drift-detection-id $$DRIFT_ID --query "StackDriftStatus" --output text)" == DRIFTED ]; then \
		echo -e "Drifted resources detected in DynamoDB stack\nPlease correct them in AWS Console"; \
		aws cloudformation describe-stack-resource-drifts --stack-name dynamodb --query "StackResourceDrifts[?StackResourceDriftStatus!='IN_SYNC']"; \
	else \
	  echo "DynamoDB stack is in sync"; \
	fi;

aerospike@deploy: ## Deploy Aerospike cluster
	helm repo add --force-update aerospike https://aerospike.github.io/aerospike-kubernetes
	helm upgrade --install $(HELM_OPTS) aerospike aerospike/aerospike \
		--version 5.5.0 \
		-f deployment/infrastructure/deployment/aerospike/values.yaml \
		--set image.repository=$(AWS_PUBLIC_ECR_REGISTRY)/$(AWS_PUBLIC_IMAGE_PREFIX)aerospike \
		$(if $(strip $(subst basic,,$(AEROSPIKE_VARIANT))),-f deployment/infrastructure/deployment/aerospike/$(AEROSPIKE_VARIANT)-values.yaml,) \
		--set-file confFilePath=$(if $(wildcard deployment/infrastructure/deployment/aerospike/$(AEROSPIKE_VARIANT)-aerospike.template.conf),deployment/infrastructure/deployment/aerospike/$(AEROSPIKE_VARIANT)-aerospike.template.conf,deployment/infrastructure/deployment/aerospike/aerospike.template.conf) \
		$(if $(strip $(subst basic,,$(AEROSPIKE_VARIANT))),--set dbReplicas=$(AEROSPIKE_NODE_COUNT),)
	kubectl apply --server-side=true -k deployment/infrastructure/deployment/aerospike/monitoring

aerospike@deploy-e2e: ## Deploy Aerospike cluster for e2e tests
	helm repo add --force-update aerospike https://aerospike.github.io/aerospike-kubernetes
	helm upgrade --install $(HELM_OPTS) aerospike-e2e aerospike/aerospike \
		--version 5.5.0 \
		-f deployment/infrastructure/deployment/aerospike/values.yaml \
		-f deployment/infrastructure/deployment/aerospike/e2e-values.yaml \
		--set-file confFilePath=deployment/infrastructure/deployment/aerospike/e2e-aerospike.template.conf

aerospike@wait: TIMEOUT=600s
aerospike@wait: ## Wait until the Aerospike cluster is ready
	kubectl rollout status --watch --timeout=$(TIMEOUT) statefulset/aerospike-aerospike

aerospike@cleanup: ## Remove Aerospike cluster (does not remove storage)
	-helm uninstall aerospike

aerospike@cleanup-storage: aerospike@cleanup ## Removes Aerospike storage resources (does not remove snapshots on AWS)
	kubectl delete pvc,pv,volumesnapshot,volumesnapshotcontent -l app=aerospike

aerospike@snapshot-create: ## Create snapshot from Aerospike cluster storage
	$(MAKE) snapshot@create SNAPSHOT_APP=aerospike SNAPSHOT_PVC_PREFIX=data-aerospike SNAPSHOT_NAME=$(SNAPSHOT_NAME)

aerospike@snapshot-restore: ## Restore Aerospike cluster storage from snapshot SNAPSHOT_NAME
	$(MAKE) snapshot@restore SNAPSHOT_APP=aerospike SNAPSHOT_PVC_PREFIX=data-aerospike SNAPSHOT_NAME=$(SNAPSHOT_NAME)

aerospike@tool: ## Run Aerospike TOOL in cluster (aql, asadm, ...)
	kubectl run at-$(shell date +%s) --rm -i -t \
		--image aerospike/aerospike-tools \
		--overrides='{"apiVersion": "v1", "spec": {"nodeSelector": { "pool": "basic-x86" }}}' \
		-- $(if $(TOOL),$(TOOL),aql) \
		--host $(or $(AEROSPIKE_HOST),aerospike-aerospike) \
		--no-config-file

aerospike@datagen: DATAGEN_ITEMS_PER_JOB ?= 10000000
aerospike@datagen: DATAGEN_DEVICES_ITEMS_PER_JOB ?= 10000000
aerospike@datagen: DATAGEN_DEVICES_PARALLELISM ?= 100
aerospike@datagen: DATAGEN_CONCURRENCY ?= 32
aerospike@datagen: export IMAGE_PREFIX=$(STACK_NAME)-
aerospike@datagen: ## Run datagen to load data on the Aerospike cluster
	envsubst < tools/datagen/deployments/k8s/generate_campaigns_aerospike.yaml.tmpl | kubectl apply -f -
	envsubst < tools/datagen/deployments/k8s/generate_audiences_aerospike.yaml.tmpl | kubectl apply -f -
	envsubst < tools/datagen/deployments/k8s/generate_budgets_aerospike.yaml.tmpl | kubectl apply -f -
	for i in $$(seq $(DATAGEN_DEVICES_PARALLELISM)) ; do \
		LOW=$$(expr \( $$i - 1 \) \* $(DATAGEN_DEVICES_ITEMS_PER_JOB)) HIGH=$$(expr $$i \* $(DATAGEN_DEVICES_ITEMS_PER_JOB)) NAME=$$i \
		envsubst < tools/datagen/deployments/k8s/generate_devices_aerospike.yaml.tmpl | kubectl apply -f - ; \
	done
	for job in generate-campaigns-aerospike generate-budgets-aerospike generate-audiences-aerospike; do \
		kubectl wait --for=condition=Complete --timeout=-1s job/$$job && kubectl delete job $$job ; \
	done
	for i in $$(seq $(DATAGEN_DEVICES_PARALLELISM)) ; do \
		kubectl wait --for=condition=Complete --timeout=-1s job/generate-devices-aerospike-$$i && \
		kubectl delete job generate-devices-aerospike-$$i ; \
	done

dynamodb@datagen: DATAGEN_ITEMS_PER_JOB ?= 10000000
dynamodb@datagen: DATAGEN_DEVICES_ITEMS_PER_JOB ?= 10000000
dynamodb@datagen: DATAGEN_DEVICES_PARALLELISM ?= 100
dynamodb@datagen: DATAGEN_CONCURRENCY ?= 32
dynamodb@datagen: export IMAGE_PREFIX=$(STACK_NAME)-
#dynamodb@datagen: DYNAMODB_TABLENAME_PREFIX=$(STACK_NAME_)
dynamodb@datagen: ## Run datagen to load data on the Aerospike cluster
	echo $(DATAGEN_ITEMS_PER_JOB)
	echo $(DYNAMODB_TABLENAME_PREFIX)
	envsubst < tools/datagen/deployments/k8s/generate_campaigns.yaml.tmpl | kubectl apply -f -
	envsubst < tools/datagen/deployments/k8s/generate_audiences.yaml.tmpl | kubectl apply -f -
	envsubst < tools/datagen/deployments/k8s/generate_budgets.yaml.tmpl | kubectl apply -f -
	for i in $$(seq $(DATAGEN_DEVICES_PARALLELISM)) ; do \
		LOW=$$(expr \( $$i - 1 \) \* $(DATAGEN_DEVICES_ITEMS_PER_JOB)) HIGH=$$(expr $$i \* $(DATAGEN_DEVICES_ITEMS_PER_JOB)) NAME=$$i \
		envsubst < tools/datagen/deployments/k8s/generate_devices.yaml.tmpl | kubectl apply -f - ; \
	done
	for job in generate-campaigns generate-budgets generate-audiences; do \
		kubectl wait --for=condition=Complete --timeout=-1s job/$$job && kubectl delete job $$job ; \
	done
	for i in $$(seq $(DATAGEN_DEVICES_PARALLELISM)) ; do \
		kubectl wait --for=condition=Complete --timeout=-1s job/generate-devices-$$i && \
		kubectl delete job generate-devices-$$i ; \
	done

cfn_nag@run: ## Run cfn_nag to find security issues in the Cloudformation stacks.
	$(CFN_NAG_RUN) \
		--input-path='/templates' \
		--template-pattern '(infrastructure/templates/.*\.yaml|infrastructure/ci/.*-stack\.yaml)' \
		$(OPTS)

cfn_nag@codekit:
cfn_nag@codekit: ## Run cfn_nag to find security issues in the Cloudformation stacks.
	$(CFN_NAG_RUN) \
		--input-path='/templates' \
		--template-pattern '(deployment/infrastructure/codekit\.yaml|deployment/infrastructure/dynamodb\.yaml|deployment/infrastructure/application\.yaml|deployment/infrastructure/serverless/kinesis-autoscale/template\.yaml|deployment/infrastructure/serverless/dax-autoscale/template\.yaml|deployment/infrastructure/templates/vpc\.yaml|deployment/infrastructure/templates/vpc-endpoints\.yaml|deployment/infrastructure/templates/eks\.yaml|deployment/infrastructure/templates/dax\.yaml|deployment/infrastructure/templates/kinesis\.yaml|deployment/infrastructure/serverless/dax-autoscale/template\.yaml)' \
		$(OPTS)

security@lint: cfn_nag@run
security@lint: viperlight@run
security@lint: ## Run the required security linters

censor@lint:  ## Delete all references to our AWS account ID from the working tree
	git grep -lz '$(AWS_ACCOUNT)' | xargs -0 sed -i 's/$(AWS_ACCOUNT)/\$${AWS_ACCOUNT}/'
