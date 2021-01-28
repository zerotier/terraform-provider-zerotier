TEST?=$$(go list ./... | grep -v 'vendor')
HOSTNAME=zerotier.com
NAMESPACE=dev
NAME=zerotier
BINARY=terraform-provider-${NAME}
VERSION=0.2
OS_ARCH=darwin_amd64
GOLANGCI_LINT_VERSION := 1.34.1

default: install

build:
	go build -o ${BINARY}

release:
	GOOS=darwin GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_darwin_amd64
	GOOS=freebsd GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_freebsd_386
	GOOS=freebsd GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_freebsd_amd64
	GOOS=freebsd GOARCH=arm go build -o ./bin/${BINARY}_${VERSION}_freebsd_arm
	GOOS=linux GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_linux_386
	GOOS=linux GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_linux_amd64
	GOOS=linux GOARCH=arm go build -o ./bin/${BINARY}_${VERSION}_linux_arm
	GOOS=openbsd GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_openbsd_386
	GOOS=openbsd GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_openbsd_amd64
	GOOS=solaris GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_solaris_amd64
	GOOS=windows GOARCH=386 go build -o ./bin/${BINARY}_${VERSION}_windows_386
	GOOS=windows GOARCH=amd64 go build -o ./bin/${BINARY}_${VERSION}_windows_amd64

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

test: 
	go test -i $(TEST) || exit 1												   
	echo $(TEST) | xargs -t -n4 go test $(TESTARGS) -timeout=30s -parallel=4					

testacc: 
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 120m   

fmt:
	go fmt ./...
	terraform fmt -recursive .

lint: bin/golangci-lint
	bin/golangci-lint run -v

reflex-lint: bin/reflex
	bin/reflex -r '\.go$$' make lint

reflex-test: bin/reflex
	bin/reflex -r '\.go$$' make test

bin/golangci-lint:
	mkdir -p bin
	wget -O- https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_LINT_VERSION)/golangci-lint-$(GOLANGCI_LINT_VERSION)-linux-amd64.tar.gz | tar vxz --strip-components=1 -C bin golangci-lint-$(GOLANGCI_LINT_VERSION)-linux-amd64/golangci-lint

bin/reflex:
	GO111MODULE=off GOBIN=${PWD}/bin go get -u github.com/cespare/reflex
