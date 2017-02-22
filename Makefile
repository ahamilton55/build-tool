OUTPUT_FILE := build-tool
S3_BUCKET := "s3://katch-sre/tools/build-tool"
REPO_ROOT := $(shell git rev-parse --show-toplevel)
GOPATH := $(GOPATH):${REPO_ROOT}
#OLD_GOPATH := $(shell echo $$GOPATH)
#BUILD_DATE := ${BUILD_DATE}

build:
	go build -o ${OUTPUT_FILE}

build_linux:
	docker run --rm -v "${PWD}":/usr/src/build_tool -w /usr/src/build_tool -e GOPATH=/usr golang:1.7 go build -o ${OUTPUT_FILE}

build_darwin:
	docker run --rm -v "${PWD}":/usr/src/build_tool -w /usr/src/build_tool -e GOPATH=/usr -e GOOS=darwin -e GOARCH=amd64 golang:1.7 go build -o ${OUTPUT_FILE}

build_windows:
	docker run --rm -v "${PWD}":/usr/src/build_tool -w /usr/src/build_tool -e GOPATH=/usr -e GOOS=windows -e GOARCH=amd64 golang:1.7 go build -o ${OUTPUT_FILE}

push_linux:
	aws s3 cp --acl public-read build-tool ${S3_BUCKET}/linux/build-tool-${BUILD_DATE}
	aws s3 cp --acl public-read build-tool ${S3_BUCKET}/linux/build-tool

push_darwin:
	aws s3 cp --acl public-read build-tool ${S3_BUCKET}/darwin/build-tool-${BUILD_DATE}
	aws s3 cp --acl public-read build-tool ${S3_BUCKET}/darwin/build-tool

push_windows:
	aws s3 cp --acl public-read build-tool ${S3_BUCKET}/windows/build-tool-${BUILD_DATE}
	aws s3 cp --acl public-read build-tool ${S3_BUCKET}/windows/build-tool

install:
	cp ./${OUTPUT_FILE} /usr/local/bin/${OUTPUT_FILE}

sudo_install:
	sudo cp ./${OUTPUT_FILE} /usr/local/bin/${OUTPUT_FILE}

update_vendor:
	@echo ${GOPATH}
	govendor add +external

install_govendor:
	go get -u -u github.com/kardianos/govendor
