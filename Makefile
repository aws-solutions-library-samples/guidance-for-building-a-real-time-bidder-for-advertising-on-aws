include config.mk

.CONTEXT_VARS := AWS_ACCOUNT AWS_REGION STACK_NAME REPO_BRANCH REPO_OWNER TARGET
$(foreach v, $(.CONTEXT_VARS), $(info $(v) = $($(v))))
.PHONY: help precommit

help: ## Help for all commands
help: _cmd_prefix= [^_]+
help: _help

_help: ## base for help command [requires "_cmd_prefix" to be set]
	@awk 'BEGIN {FS = ":.*?## "} \
	/^$(_cmd_prefix)[0-9a-zA-Z_-]+$(_cmd_suffix):.*?##/ \
	{printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | \
	sed -E 's/@([0-9a-zA-Z_-]+)/\o033[31m@\o033[0m\o033[33m\1\o033[0m/g' | \
	uniq --group -w 20

# List various directories such that we exclude the huge and poorly ZIP-compressible docs/benchmarks/results.
dist: ARCHIVE_PATH?=bidder.zip
dist: ## Export the project to a ZIP.
	git archive --format=zip -o $(ARCHIVE_PATH) HEAD $$( \
		git ls-tree -r HEAD | cut -f 2- | \
		grep -v \
			-e '^docs/benchmarks/results' \
			-e '^tools/benchmark-utils/scenarios/.*/outputs' \
			-e '^deployment/infrastructure/ci/.*' \
			-e '^deployment/infrastructure/\(bidder-benchmarkaerospike\|chatbot\|cf\|dns\|docs\|dynamodb-e2e\|dynamodb-rich\|dynamodb-tiny\|dynamodb\|iam\).yaml' \
	)

include apps/bidder/tools/Makefile
include apps/model/tools/Makefile
include deployment/infrastructure/Makefile
include cdk/pipeline/Makefile
include tools/*/Makefile

# there must be empty line of the end (include cannot be last line) of file for autocomplete to work
