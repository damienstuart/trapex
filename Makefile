
BUILDARCH=x86_64
TARGET=trapex
image = alpine:3.15
docker_tag_trapex = damienstuart/trapex
container_trapex = trapex
configuration_path_trapex = /Users/kellskearney/go/src/trapex/tools
docker_tag_clickhouse = damienstuart/clickhouse
container_clickhouse = clickhouse
#configuration_path_clickhouse = /Users/kellskearney/go/src/trapex/tools


build:
	go build

build_all: plugins build

plugins:
	cd txPlugins && ./build_plugins.sh

deps:
	go get ./...

test: build
	go test
	cd txPlugins && go test

fmt:
	gofmt -w *.go
	gofmt -w txPlugins/*.go
	gofmt -w txPlugins/actions/*/*.go
	gofmt -w txPlugins/generators/*/*.go
	gofmt -w txPlugins/metrics/*/*.go
	gofmt -w cmds/*/*.go
	git commit -m "gofmt" -a

rpm: build
	rpmbuild -ba tools/rpm.spec

clean: clean_plugins
	rm -rf ~/rpmbuild/BUILD/${TARGET} ~/rpmbuild/BUILD/${BUILDARCH}/*
	go clean

clean_plugins:
	find txPlugins -name \*.so -delete

install:
	cd ~/rpmbuild/RPMS/${BUILDARCH} && sudo yum install -y `ls -1rt | tail -1`

push:
	git push -u origin $(shell git symbolic-ref --short HEAD)

# ----  Docker: trapex  ----------------------------
.PHONY: trapex
trapex:
	DOCKER_BUILD=0 docker build -t $(docker_tag_trapex) -f tools/docker/Dockerfile .

trapex_aws:
	DOCKER_BUILD=0 docker build -t $(docker_tag_trapex) -f tools/docker/Dockerfile.amazonlinux .

run:
	docker run --name $(container_trapex) -v $(configuration_path):/opt/trapex/etc -p 162:162 -p 5080:80 $(docker_tag_trapex)

stop:
	docker stop $(container_trapex)
	docker rm $(container_trapex)

pull:
	docker pull $(image)

# ----  Docker: clickhouse  ----------------------------
clickhouse:
	DOCKER_BUILD=0 docker build -t $(docker_tag_clickhouse) -f tools/docker/Dockerfile.clickhouse .

run_click:
	docker run --name $(container_clickhouse) -v $(configuration_path):/opt/trapex/etc -p 162:162 -p 5080:80 $(docker_tag_clickhouse)

stop_click:
	docker stop $(container_trapex)
	docker rm $(container_trapex)

# ----  AWS  ----------------------------
codebuild:
# Need to run the following first
# aws configure
	#aws cloudformation deploy --template-file tools/aws/codebuild_cfn.yml --stack-name trapexrpm --capabilities CAPABILITY_IAM
	aws cloudformation deploy --template-file tools/aws/codebuild_docker.yml --stack-name trapexdocker --capabilities CAPABILITY_IAM
	#aws cloudformation deploy --template-file tools/aws/codebuild_batch_cfn.yml --stack-name trapexbatchrpm --capabilities CAPABILITY_IAM --parameter-overrides StreamId=rpm BuildSpec=tools/aws/buildspec_batch_rpm.yml
	#aws cloudformation deploy --template-file tools/aws/codebuild_batch_cfn.yml --stack-name trapexbatchnopkg --capabilities CAPABILITY_IAM --parameter-overrides StreamId=nopkg BuildSpec=tools/aws/buildspec_batch_nopkg.yml CodeBuildImage=aws/codebuild/standard:5.0 

