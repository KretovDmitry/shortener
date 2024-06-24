MODULE = $(shell go list -m)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || echo "1.0.0")
PACKAGES := $(shell go list ./... | grep -v /vendor/)
SECRET_FILE ?= ./secret.yml
APP_DSN ?= $(shell sed -n 's/^dsn:[[:space:]]*"\(.*\)"/\1/p' $(SECRET_FILE))
LDFLAGS := -ldflags "\
			-X main.buildVersion=${VERSION}\
			-X 'main.buildDate=$(shell date +'%Y/%m/%d %H:%M:%S')'\
			-X main.buildCommit=$(shell git rev-parse --short HEAD)"

PID_FILE := './.pid'
FSWATCH_FILE := './fswatch.cfg'
MAIN_FILE := './cmd/shortener/main.go'
LINT_FILE := './cmd/staticlint/main.go'
LOCAL_CONFIG := './config/local.yml'
BINARY_PATH := './cmd/shortener/shortener'
DISABLE_HTTPS := 'ENABLE_HTTPS=false'

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
	CGO_ENABLED=0 go build ${LDFLAGS} -a -o ${BINARY_PATH} $(MODULE)/cmd/shortener

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

.PHONY: yp-statictest
yp-statictest: ## run Yandex Practicum static analysis tool
	@chmod +x ./statictest
	@go vet -vettool=./statictest ./...

.PHONY: yp-test
yp-test: ## run all Yandex Practicum E2E tests
	make build
	make yp-test-iter1
	make yp-test-iter2
	make yp-test-iter3
	make yp-test-iter4
	make yp-test-iter5
	make yp-test-iter6
	make yp-test-iter7
	make yp-test-iter8
	make yp-test-iter9
	make yp-test-iter10
	## make yp-test-iter11
	make yp-test-iter12

.PHONY: yp-test-iter1
yp-test-iter1: ## run test for iter1 [sudo]
	@chmod +x ./shortenertestbeta
	@echo "------------- Running TestIteration1 -------------"
	@sudo CONFIG=${LOCAL_CONFIG} ${DISABLE_HTTPS} \
		./shortenertestbeta -test.v -test.run=^TestIteration1$$ \
		-binary-path=${BINARY_PATH}

.PHONY: yp-test-iter2
yp-test-iter2: ## run test for iter2 [sudo]
	@chmod +x ./shortenertestbeta
	@echo "------------- Running TestIteration2 -------------"
	@sudo CONFIG=${LOCAL_CONFIG} ${DISABLE_HTTPS} \
		./shortenertestbeta -test.v -test.run=^TestIteration2$$ \
		-source-path=.

.PHONY: yp-test-iter3
yp-test-iter3: ## run test for iter3 [sudo]
	@chmod +x ./shortenertestbeta
	@echo "------------- Running TestIteration3 -------------"
	@sudo CONFIG=${LOCAL_CONFIG} ${DISABLE_HTTPS} \
		./shortenertestbeta -test.v -test.run=^TestIteration3$$ \
		-source-path=.

.PHONY: yp-test-iter4
yp-test-iter4: ## run test for iter4 [sudo]
	@chmod +x ./shortenertestbeta
	@echo "------------- Running TestIteration4 -------------"
	@sudo CONFIG=${LOCAL_CONFIG} ${DISABLE_HTTPS} \
		./shortenertestbeta -test.v -test.run=^TestIteration4$$ \
		-binary-path=${BINARY_PATH} -server-port=5000

.PHONY: yp-test-iter5
yp-test-iter5: ## run test for iter5 [sudo]
	@chmod +x ./shortenertestbeta
	@echo "------------- Running TestIteration5 -------------"
	@sudo CONFIG=${LOCAL_CONFIG} ${DISABLE_HTTPS} \
		./shortenertestbeta -test.v -test.run=^TestIteration5$$ \
		-binary-path=${BINARY_PATH} -server-port=5000

.PHONY: yp-test-iter6
yp-test-iter6: ## run test for iter6 [sudo]
	@chmod +x ./shortenertestbeta
	@echo "------------- Running TestIteration6 -------------"
	@sudo CONFIG=${LOCAL_CONFIG} ${DISABLE_HTTPS} \
		./shortenertestbeta -test.v -test.run=^TestIteration6$$ \
		-source-path=.

.PHONY: yp-test-iter7
yp-test-iter7: ## run test for iter7 [sudo]
	@chmod +x ./shortenertestbeta
	@echo "------------- Running TestIteration7 -------------"
	@sudo CONFIG=${LOCAL_CONFIG} ${DISABLE_HTTPS} \
		./shortenertestbeta -test.v -test.run=^TestIteration7$$ \
		-binary-path=${BINARY_PATH} -source-path=.

.PHONY: yp-test-iter8
yp-test-iter8: ## run test for iter8 [sudo]
	@chmod +x ./shortenertestbeta
	@echo "------------- Running TestIteration8 -------------"
	@sudo CONFIG=${LOCAL_CONFIG} ${DISABLE_HTTPS} \
		./shortenertestbeta -test.v -test.run=^TestIteration8$$ \
		-binary-path=${BINARY_PATH}

.PHONY: yp-test-iter9
yp-test-iter9: ## run test for iter9 [sudo]
	@chmod +x ./shortenertestbeta
	@echo "------------- Running TestIteration9 -------------"
	@sudo CONFIG=${LOCAL_CONFIG} ${DISABLE_HTTPS} \
		./shortenertestbeta -test.v -test.run=^TestIteration9$$ \
		-binary-path=${BINARY_PATH} -source-path=. \
		-file-storage-path=/tmp/short-url-db.json

.PHONY: yp-test-iter10
yp-test-iter10: ## run test for iter10 [sudo]
	@chmod +x ./shortenertestbeta
	@echo "------------- Running TestIteration10 -------------"
	@sudo CONFIG=${LOCAL_CONFIG} ${DISABLE_HTTPS} \
		./shortenertestbeta -test.v -test.run=^TestIteration10$$ \
		-binary-path=${BINARY_PATH} -source-path=. \
		-database-dsn=${APP_DSN}

.PHONY: yp-test-iter11
yp-test-iter11: ## run test for iter11 [sudo]
	@chmod +x ./shortenertestbeta
	@echo "------------- Running TestIteration11 -------------"
	@sudo CONFIG=${LOCAL_CONFIG} ${DISABLE_HTTPS} \
		./shortenertestbeta -test.v -test.run=^TestIteration11$$ \
		-binary-path=${BINARY_PATH} -database-dsn=${APP_DSN}

.PHONY: yp-test-iter12
yp-test-iter12: ## run test for iter12 [sudo]
	@chmod +x ./shortenertestbeta
	@echo "------------- Running TestIteration12 -------------"
	@sudo CONFIG=${LOCAL_CONFIG} ${DISABLE_HTTPS} \
		./shortenertestbeta -test.v -test.run=^TestIteration12$$ \
		-binary-path=${BINARY_PATH} -database-dsn=${APP_DSN}

