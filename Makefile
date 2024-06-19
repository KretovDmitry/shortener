MODULE = $(shell go list -m)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || echo "1.0.0")
PACKAGES := $(shell go list ./... | grep -v /vendor/)
LDFLAGS := -ldflags "\
			-X main.buildVersion=${VERSION}\
			-X 'main.buildDate=$(shell date +'%Y/%m/%d %H:%M:%S')'\
			-X main.buildCommit=$(shell git rev-parse --short HEAD)"

PID_FILE := './.pid'
FSWATCH_FILE := './fswatch.cfg'
MAIN_FILE := 'cmd/shortener/main.go'
LINT_FILE := 'cmd/staticlint/main.go'

.PHONY: default
default: help

# generate help info from comments
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help: ## help information about make commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: test
test: ## run unit tests
	@echo "mode: count" > coverage-all.out
	@$(foreach pkg,$(PACKAGES), \
		go test -p=1 -cover -covermode=count -coverprofile=coverage.out ${pkg}; \
		tail -n +2 coverage.out >> coverage-all.out;)

.PHONY: test-cover
test-cover: test ## run unit tests and show test coverage information
	go tool cover -html=coverage-all.out -o coverage.html

.PHONY: run
run: ## run the API server
	@go run ${LDFLAGS} ${MAIN_FILE}

.PHONY: run-restart
run-restart: ## restart the API server
	@pkill -P `cat $(PID_FILE)` || true
	@printf '%*s\n' "80" '' | tr ' ' -
	@echo "Source file changed. Restarting server..."
	@go run ${LDFLAGS} ${MAIN_FILE} & echo $$! > $(PID_FILE)
	@printf '%*s\n' "80" '' | tr ' ' -

run-live: ## run the API server with live reload support (requires fswatch)
	@go run ${LDFLAGS} ${MAIN_FILE} & echo $$! > $(PID_FILE)
	@fswatch -x -o --event Created --event Updated --event Renamed -r internal pkg cmd config | xargs -I {} make run-restart

.PHONY: build
build:  ## build the API server binary
	CGO_ENABLED=0 go build ${LDFLAGS} -a -o cmd/shortener/shortener $(MODULE)/cmd/shortener

.PHONY: clean
clean: ## remove temporary files
	rm -rf server coverage.out coverage-all.out coverage.html

.PHONY: version
version: ## display the version of the API server
	@echo $(VERSION)

.PHONY: lint-custom
lint-custom: ## run custom static analysis tool
	go build -o cmd/staticlint/lint ${LINT_FILE}
	$(shell ./cmd/staticlint/lint ./...)

.PHONY: lint-ci
lint-ci: ## run golangchi lint on all Go packages
	docker run -t --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v1.59.1 golangci-lint run -v


.PHONY: mock
mock: ## generate all mocks for the project with mockgen
	make mock-store

.PHONY: mock-store
mock-store: ## generate mock store with mockgen
	mockgen -destination=mocks/mock_store.go -package=mocks github.com/KretovDmitry/shortener/internal/db URLStorage
