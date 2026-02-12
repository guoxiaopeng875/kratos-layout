GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)

INTERNAL_PROTO_FILES=$(shell find internal -name *.proto)
API_PROTO_FILES=$(shell find api -name *.proto)


.PHONY: init
# init env
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

.PHONY: config
# generate internal proto
config:
	protoc --proto_path=./internal \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:./internal \
	       $(INTERNAL_PROTO_FILES)

.PHONY: api
# generate api proto
api:
	protoc --proto_path=./api \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:./api \
 	       --go-http_out=paths=source_relative:./api \
 	       --go-grpc_out=paths=source_relative:./api \
	       --openapi_out=fq_schema_naming=true,default_response=false:. \
	       $(API_PROTO_FILES)

.PHONY: build
# build
build:
	mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...

# build single binary for CICD (usage: make bin/${APP_NAME})
bin/%:
	@./scripts/build.sh "$*" "$(VERSION)"

.PHONY: generate
# generate
generate:
	go generate ./...
	go mod tidy

.PHONY: all
# generate all
all:
	make api
	make config
	make generate

.PHONY: test
# run unit tests (excludes integration tests)
test:
	go test -v -race $$(go list ./... | grep -v /test/integration/)

.PHONY: test-integration
# run integration tests
test-integration:
	go test -v -race -timeout 15m ./test/integration/...

.PHONY: check
# format code, run tests and lint
check:
	goimports -w .
	gofmt -w .
	go test -race $$(go list ./... | grep -v /test/integration/)
	golangci-lint run ./...

.PHONY: lint
# run linter
lint:
	golangci-lint run ./...

.PHONY: fmt
# format code
fmt:
	goimports -w .
	gofmt -w .

.PHONY: hooks
# install git hooks
hooks:
	cp scripts/pre-commit.sh .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "Git hooks installed."


.PHONY: init-db
# initialize database schema
init-db:
	@echo "Dropping existing test database..."
	docker exec -i kratos-mysql mysql -uroot -proot -e "DROP DATABASE IF EXISTS app_local;"
	@echo "Creating database..."
	docker exec -i kratos-mysql mysql -uroot -proot -e "CREATE DATABASE app_local CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
	@echo "Creating tables..."
	cat scripts/sql/migration/*.sql | docker exec -i kratos-mysql mysql -uroot -proot app_local
	@echo "Initializing test data..."
	#cat deploy/local/init_data.sql | docker exec -i kratos-mysql mysql -uroot -proot app_local
	@echo "Test database initialized!"

.PHONY: migrate-diff
# generate migration SQL from GORM model changes
migrate-diff:
	atlas migrate diff --env local

.PHONY: migrate-hash
# rehash migration directory after manual edits
migrate-hash:
	atlas migrate hash --env local

.PHONY: reset-db
# reset database (drop and recreate)
reset-db:
	@echo "Clearing Redis cache..."
	docker exec -i kratos-redis redis-cli FLUSHALL || true
	$(MAKE) init-db

.PHONY: run
# start all services
run:
	./scripts/start-app-service.sh start

.PHONY: stop
# stop all services
stop:
	./scripts/start-app-service.sh stop || true

.PHONY: rebuild
# rebuild and start all services
rebuild:
	./scripts/start-app-service.sh rebuild

.PHONY: reset
# reset test environment (stop services, reinit db, start services)
reset:
	$(MAKE) stop
	$(MAKE) reset-db
	$(MAKE) run

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
