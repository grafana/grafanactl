# Within devbox
ifneq "$(DEVBOX_CONFIG_DIR)" ""
    RUN_DEVBOX:=
else # Normal shell
    RUN_DEVBOX:=devbox run
endif

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: lint
lint: check-binaries ## Lints the code base.
	$(RUN_DEVBOX) golangci-lint run -c .golangci.yaml

.PHONY: tests
tests: check-binaries ## Runs the tests.
	$(RUN_DEVBOX) go test -v ./...

.PHONY: deps
deps: check-binaries ## Installs the dependencies.
	$(RUN_DEVBOX) go mod vendor
	$(RUN_DEVBOX) pip install -qq -r requirements.txt

.PHONY: docs
docs: check-binaries ## Generates the documentation.
	$(RUN_DEVBOX) mkdocs build -f mkdocs.yml -d ./build/documentation

.PHONY: serve-docs
serve-docs: check-binaries ## Serves the documentation and watches for changes.
	$(RUN_DEVBOX) mkdocs serve -f mkdocs.yml 

.PHONY: check-binaries
check-binaries: ## Check that the required binaries are present.
	@devbox version >/dev/null 2>&1 || (echo "ERROR: devbox is required. See https://www.jetify.com/devbox/docs/quickstart/"; exit 1)
