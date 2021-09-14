HOSTNAME=registry.terraform.io
NAMESPACE=zerotier
NAME=zerotier
BINARY=terraform-provider-${NAME}
VERSION=0.2.0
OS_ARCH=$(shell go env GOOS)_$(shell go env GOARCH)
GOLANGCI_LINT_VERSION=1.34.1
BUILD=go build -ldflags "-X github.com/zerotier/terraform-provider-zerotier/pkg/zerotier.Version=${VERSION}" -o

ifeq ($(QUIET_TESTS),)
TEST_VERBOSE = -v
endif

ifneq ($(FORCE_TESTS),)
TEST_COUNT = -count 1
else 
TEST_COUNT = 
endif

ifneq ($(TEST),)
RUN_TEST=-run "$(TEST)"
else
RUN_TEST=
endif

default: install

checks: fmt lint docs test

mktfrc:
	@echo Creating bootstrap terraform rc file in test.tfrc...
	sh mktfrc.sh

build:
	${BUILD} ${BINARY}

release:
	GOOS=darwin GOARCH=amd64 ${BUILD} ./bin/${BINARY}_${VERSION}_darwin_amd64
	GOOS=freebsd GOARCH=386 ${BUILD} ./bin/${BINARY}_${VERSION}_freebsd_386
	GOOS=freebsd GOARCH=amd64 ${BUILD} ./bin/${BINARY}_${VERSION}_freebsd_amd64
	GOOS=freebsd GOARCH=arm ${BUILD} ./bin/${BINARY}_${VERSION}_freebsd_arm
	GOOS=linux GOARCH=386 ${BUILD} ./bin/${BINARY}_${VERSION}_linux_386
	GOOS=linux GOARCH=amd64 ${BUILD} ./bin/${BINARY}_${VERSION}_linux_amd64
	GOOS=linux GOARCH=arm ${BUILD} ./bin/${BINARY}_${VERSION}_linux_arm
	GOOS=openbsd GOARCH=386 ${BUILD} ./bin/${BINARY}_${VERSION}_openbsd_386
	GOOS=openbsd GOARCH=amd64 ${BUILD} ./bin/${BINARY}_${VERSION}_openbsd_amd64
	GOOS=solaris GOARCH=amd64 ${BUILD} ./bin/${BINARY}_${VERSION}_solaris_amd64
	GOOS=windows GOARCH=386 ${BUILD} ./bin/${BINARY}_${VERSION}_windows_386
	GOOS=windows GOARCH=amd64 ${BUILD} ./bin/${BINARY}_${VERSION}_windows_amd64

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

fmt:
	go fmt ./...
	terraform fmt -recursive .

test-build:
	${BUILD} .tfdata/registry.terraform.io/hashicorp/zerotier/${VERSION}/${OS_ARCH}/${BINARY}

test: test-image mktfrc test-build
	go test ${TEST_VERBOSE} ./... ${TEST_COUNT} ${RUN_TEST} -p 1 # one test at at time

lint: bin/golangci-lint
	bin/golangci-lint run -v

reflex-lint: bin/reflex
	bin/reflex -r '\.go$$' make lint

reflex-build: bin/reflex
	bin/reflex -r '\.go$$' -- go build ./...

reflex-test: bin/reflex
	bin/reflex -r '\.(go|tf)$$' make test

bin/golangci-lint:
	mkdir -p bin
	wget -O- https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_LINT_VERSION)/golangci-lint-$(GOLANGCI_LINT_VERSION)-linux-amd64.tar.gz | tar vxz --strip-components=1 -C bin golangci-lint-$(GOLANGCI_LINT_VERSION)-linux-amd64/golangci-lint

bin/reflex:
	GO111MODULE=off GOBIN=${PWD}/bin go get -u github.com/cespare/reflex

ifdef NOCACHE
NOCACHE_FLAG=--no-cache
else
NOCACHE_FLAG=
endif

test-image:
	docker build ${NOCACHE_FLAG} --pull -t zerotier/terraform-test .

docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

.PHONY: docs
