
BUILDARCH=x86_64
TARGET=trapex
image = alpine:3.15
docker_tag = damientstuart/trapex
container = trapex
configuration_path = /Users/kellskearney/go/src/trapex/tools


build:
	go build ./...

deps:
	go get ./...

test: build
	go test

rpm: build
	rpmbuild -ba tools/rpm.spec

clean:
	rm -rf ~/rpmbuild/BUILD/${TARGET} ~/rpmbuild/BUILD/${BUILDARCH}/*

install:
	cd ~/rpmbuild/RPMS/${BUILDARCH} && sudo yum install -y `ls -1rt | tail -1`

push:
	git push -u origin $(shell git symbolic-ref --short HEAD)

# ----  Docker  ----------------------------
docker:
	DOCKER_BUILD=0 docker build -t $(docker_tag) -f tools/docker/Dockerfile .

run:
	docker run --name $(container) -v $(configuration_path):/opt/trapex/etc -p 162:162 -p 5080:80 $(docker_tag)

stop:
	docker stop $(container)
	docker rm $(container)

pull:
	docker pull $(image)

# ----  AWS  ----------------------------
codebuild:
# Need to run the following first
# aws configure
	#aws cloudformation deploy --template-file tools/aws/codebuild_cfn.yml --stack-name trapexrpm --capabilities CAPABILITY_IAM
	aws cloudformation deploy --template-file tools/aws/codebuild_docker.yml --stack-name trapexdocker --capabilities CAPABILITY_IAM
	#aws cloudformation deploy --template-file tools/aws/codebuild_batch_cfn.yml --stack-name trapexbatchrpm --capabilities CAPABILITY_IAM --parameter-overrides StreamId=rpm BuildSpec=tools/aws/buildspec_batch_rpm.yml
	#aws cloudformation deploy --template-file tools/aws/codebuild_batch_cfn.yml --stack-name trapexbatchnopkg --capabilities CAPABILITY_IAM --parameter-overrides StreamId=nopkg BuildSpec=tools/aws/buildspec_batch_nopkg.yml CodeBuildImage=aws/codebuild/standard:5.0 

