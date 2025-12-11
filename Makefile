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
	rm -f ./dist/arklite*

local-build:
	go build ${BUILDFLAGS} -v ${LDFLAGS} -o ${BINARY}

linux-amd64-build:
	CGO_ENABLED=1 \
	CC="zig cc -O3 -target x86_64-linux-musl -lc" \
	CXX="zig c++ -O3 -target x86_64-linux-musl -lc" \
	GOOS=linux GOARCH=amd64 go build -v ${BUILDFLAGS} ${LDFLAGS} -o ./dist/arklite-linux-amd64

linux-arm64-build:
	CGO_ENABLED=1 \
	CC="zig cc -O3 -target aarch64-linux-musl -lc" \
	CXX="zig c++ -O3 -target aarch64-linux-musl -lc" \
	GOOS=linux GOARCH=arm64 go build -v ${BUILDFLAGS} ${LDFLAGS} -o ./dist/arklite-linux-arm64

dist-gzip:
	cd ./dist && tar -czf arklite-linux-amd64.tar.gz arklite-linux-amd64
	cd ./dist && tar -czf arklite-linux-arm64.tar.gz arklite-linux-arm64

dist-sha256:
	cd ./dist && sha256sum arklite-linux-amd64.tar.gz > arklite-linux-amd64.tar.gz.sha256
	cd ./dist && sha256sum arklite-linux-arm64.tar.gz > arklite-linux-arm64.tar.gz.sha256

dist: clean linux-amd64-build linux-arm64-build dist-gzip dist-sha256

build: clean local-build
