BINARY=arklite
PKG=github.com/bak1an/arklite

BUILD=$(shell date +%FT%T%z)
GIT_REV=$(shell git rev-parse HEAD)
GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
GIT_TAG=$(shell git describe --tags --abbrev=0)

LDFLAGS=-ldflags "-s -w -X ${PKG}/version.build=${BUILD} -X ${PKG}/version.gitRev=${GIT_REV} -X ${PKG}/version.gitBranch=${GIT_BRANCH} -X ${PKG}/version.gitTag=${GIT_TAG}"
BUILDFLAGS=-trimpath

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
	go build ${BUILDFLAGS} -v ${LDFLAGS} -o ${BINARY}

linux-build:
	CGO_ENABLED=1 \
	CC="zig cc -O3 -target x86_64-linux-musl -lc" \
	CXX="zig c++ -O3 -target x86_64-linux-musl -lc" \
	GOOS=linux GOARCH=amd64 go build -v ${BUILDFLAGS} ${LDFLAGS} -o ${BINARY}

linux: clean linux-build

build: clean local-build
