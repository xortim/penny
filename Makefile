COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null)
GITVERSION  ?= $(shell git describe --tags --exact-match 2>/dev/null || git describe --tags 2>/dev/null || echo "v0.0.0-$(COMMIT_HASH)")

GO          ?= go
GOOS        ?= $(shell $(GO) env GOOS)
GOARCH      ?= $(shell $(GO) env GOARCH)
PACKAGENAME := $(shell go list -m -f '{{.Path}}')
GOLDFLAGS   ?= -s -w -X $(PACKAGENAME)/conf.Executable=$(EXECUTABLE) -X $(PACKAGENAME)/conf.GitVersion=$(GITVERSION)
GOBUILD     ?= CGO_ENABLED=0 $(GO) build -ldflags="$(GOLDFLAGS)"
GO_FILES    := $(shell find . -type f -name '*.go')

EXECUTABLE  := penny
ARTIFACT    := dist/$(GOOS)-$(GOARCH)/$(EXECUTABLE)

DB_PASS      ?= $(shell openssl rand -hex 16)
DB_ROOT_PASS ?= $(shell openssl rand -hex 16)

.PHONY: all
all: clean verify lint test build

###############
##@ Development

# This is to allow make to detect when other targes should be rerun (source changes)
$(GO_FILES):
	@stat -c "%y %n" "$@"

.PHONY: $(EXECUTABLE)
build: $(ARTIFACT) ## Build binary
$(ARTIFACT): $(GO_FILES)
	@$(MAKE) --no-print-directory log-build
	@$(GOBUILD) -o $@
	@ln -fs $(GOOS)-$(GOARCH)/$(EXECUTABLE) dist/$(EXECUTABLE)

.PHONY: verify
verify:   ## Verify 'vendor' dependencies
	@ $(MAKE) --no-print-directory log-$@
	$(GO) mod verify

.PHONY: container
container: ## Build container using docker
	@$(MAKE) --no-print-directory log-$@
	@docker build -t $(EXECUTABLE):local .

.PHONY: lint ## Lint the project
lint:
	@$(MAKE) --no-print-directory log-$@
	@golint

.PHONY: test
test: coverage.out ## Execute tests
coverage.out: $(GO_FILES)
	@$(MAKE) --no-print-directory log-$@
	$(GO) test -coverprofile=coverage.out -covermode=atomic -v ./...

.PHONY: clean
clean: ## Clean the workspace including modcache and dist/
	@$(MAKE) --no-print-directory log-$@
	@$(GO) clean -modcache
	@rm -rf dist/* coverage.out

.PHONY: tools
tools: ## Install tools needed for development
	@$(MAKE) --no-print-directory log-$@
	@go get -u golang.org/x/lint/golint

###############
##@ Release

.PHONY: changelog
changelog: ## Generate CHANGELOG.md using git-cliff
	@$(MAKE) --no-print-directory log-$@
	git-cliff --output CHANGELOG.md

.PHONY: snapshot
snapshot: changelog ## Build a snapshot release locally (no publish)
	@$(MAKE) --no-print-directory log-$@
	goreleaser release --snapshot --clean

.PHONY: release
release: changelog ## Create a release with goreleaser
	@$(MAKE) --no-print-directory log-$@
	goreleaser release --clean

###############
##@ Database
.PHONY: start-db
start-db: ## Start maria db - export DB_ROOT_PASS and DB_PASS to set credentials
	@$(MAKE) --no-print-directory log-$@
	docker run --name $(EXECUTABLE)-mariadb \
		-v ${HOME}/.$(EXECUTABLE)/db:/var/lib/mysql \
		-e MARIADB_ROOT_PASSWORD="${DB_ROOT_PASS}" \
		-e MARIADB_DATABASE=$(EXECUTABLE)_dev \
		-e MARIADB_USER=$(EXECUTABLE) \
		-e MARIADB_PASSWORD="${DB_PASS}" \
		-p 3306:3306 \
		-d mariadb:10.5

.PHONY: stop-db
stop-db: ## Stop maria db
	@$(MAKE) --no-print-directory log-$@
	@docker stop $(EXECUTABLE)-mariadb
	@docker rm $(EXECUTABLE)-mariadb

###########################################################################
## Self-Documenting Makefile Help and logging                            ##
## https://github.com/terraform-docs/terraform-docs/blob/master/Makefile ##
## https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html    ##
###########################################################################

########
##@ Help

.PHONY: help
help:   ## Display this help
	@awk \
		-v "col=\033[36m" -v "nocol=\033[0m" \
		' \
			BEGIN { \
				FS = ":.*##" ; \
				printf "Usage:\n  make %s<target>%s\n", col, nocol \
			} \
			/^[a-zA-Z_-]+:.*?##/ { \
				printf "  %s%-12s%s %s\n", col, $$1, nocol, $$2 \
			} \
			/^##@/ { \
				printf "\n%s%s%s\n", nocol, substr($$0, 5), nocol \
			} \
		' $(MAKEFILE_LIST)

log-%:
	@grep -h -E '^$*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk \
			'BEGIN { \
				FS = ":.*?## " \
			}; \
			{ \
				printf "\033[36m==> %s\033[0m\n", $$2 \
			}'
