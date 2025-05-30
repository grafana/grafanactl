SHELL := /bin/bash

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

.PHONY: all
all: lint tests build docs ## Lints, tests, builds, and generates the documentation.

.PHONY: lint
lint: check-binaries ## Lints the code base.
	$(RUN_DEVBOX) golangci-lint run -c .golangci.yaml

.PHONY: tests
tests: check-binaries ## Runs the tests.
	$(RUN_DEVBOX) go test -v ./...

GIT_REVISION  ?= $(shell git rev-parse --short HEAD)
GIT_VERSION   ?= $(shell git describe --tags --exact-match 2>/dev/null || echo "")
BUILD_DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
VERSION_FLAGS := -X main.version=${GIT_VERSION} -X main.commit=${GIT_REVISION} -X main.date=${BUILD_DATE}

.PHONY: build
build: check-binaries ## Builds the binary into the `./bin/grafanactl`.
	$(RUN_DEVBOX) go build \
		-buildvcs=false \
		-ldflags="${VERSION_FLAGS}" \
		-o bin/grafanactl \
		./cmd/grafanactl

.PHONY: install
install: build ## Installs the binary into `$GOPATH/bin`.
ifndef GOPATH
	@echo "GOPATH is not defined"
	exit 1
endif
	@cp "bin/grafanactl" "${GOPATH}/bin/grafanactl"

.PHONY: deps
deps: check-binaries ## Installs the dependencies.
	$(RUN_DEVBOX) go mod vendor
	$(RUN_DEVBOX) pip install -qq -r requirements.txt

.PHONY: clean
clean: ## Cleans the project.
	rm -rf bin
	rm -rf vendor
	rm -rf .devbox
	rm -rf .venv

.PHONY: check-binaries
check-binaries: ## Check that the required binaries are present.
	@devbox version >/dev/null 2>&1 || (echo "ERROR: devbox is required. See https://www.jetify.com/devbox/docs/quickstart/"; exit 1)


##@ Documentation

.PHONY: docs
docs: check-binaries reference ## Generates the documentation.
	$(RUN_DEVBOX) mkdocs build -f mkdocs.yml -d ./build/documentation

.PHONY: reference
reference: cli-reference env-var-reference config-reference ## Generates all references documentation pages.

.PHONY: reference-drift
reference-drift: cli-reference-drift env-var-reference-drift config-reference-drift ## Checks for drift in all references documentation pages.

.PHONY: cli-reference
cli-reference: check-binaries ## Generates a reference for the CLI.
	@rm -rf ./docs/reference/cli
	@$(RUN_DEVBOX) go run scripts/cmd-reference/*.go "./docs/reference/cli"

.PHONY: env-var-reference
env-var-reference: check-binaries ## Generates an environment variables reference.
	@rm -rf ./docs/reference/environment-variables
	@$(RUN_DEVBOX) go run scripts/env-vars-reference/*.go "./docs/reference/environment-variables"

.PHONY: config-reference
config-reference: check-binaries ## Generates a reference for the configuration file.
	@rm -rf ./docs/reference/configuration
	@$(RUN_DEVBOX) go run scripts/config-reference/*.go "./docs/reference/configuration"

.PHONY: cli-reference-drift
cli-reference-drift: cli-reference ## Checks for drift in the generated CLI reference.
	@if ! git diff --exit-code --quiet HEAD ./docs/reference/cli/ ; then \
		echo "Drift detected in the generated CLI reference."; \
		echo 'Run `make cli-reference` and commit the modified files.'; \
		exit 1; \
 	fi

.PHONY: env-var-reference-drift
env-var-reference-drift: env-var-reference ## Checks for drift in the generated environment variables reference.
	@if ! git diff --exit-code --quiet HEAD ./docs/reference/environment-variables/ ; then \
		echo "Drift detected in the generated environment variables reference."; \
		echo 'Run `make env-var-reference` and commit the modified files.'; \
		exit 1; \
 	fi

.PHONY: config-reference-drift
config-reference-drift: config-reference ## Checks for drift in the generated config file reference.
	@if ! git diff --exit-code --quiet HEAD ./docs/reference/configuration/ ; then \
		echo "Drift detected in the generated config reference."; \
		echo 'Run `make config-reference` and commit the modified files.'; \
		exit 1; \
 	fi

.PHONY: serve-docs
serve-docs: check-binaries ## Serves the documentation and watches for changes.
	$(RUN_DEVBOX) mkdocs serve -f mkdocs.yml
