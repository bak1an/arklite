BINARY=arklite
PKG=github.com/bak1an/arklite

BUILD=$(shell date +%FT%T%z)
GIT_REV=$(shell git rev-parse HEAD)
GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)

LDFLAGS=-ldflags "-X ${PKG}/config.build=${BUILD} -X ${PKG}/config.gitRev=${GIT_REV} -X ${PKG}/config.gitBranch=${GIT_BRANCH}"

.DEFAULT_GOAL := build

fmt:
	go fmt ./... && go tool goimports -w .

test:
	go test -v -race ./...

vet:
	go vet -v ./...

nils:
	go tool nilaway ./...

check: vet nils test

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi

local-build:
	go build -v ${LDFLAGS} -o ${BINARY}

linux-build:
	GOOS=linux GOARCH=amd64 go build -v ${LDFLAGS} -o ${BINARY}

linux: clean linux-build

build: clean local-build
