
BUILDARCH=x86_64
TARGET=trapex

build:
	go build ./...

deps:
	go get ./...

rpm: build
	rpmbuild -ba tools/rpm.spec

clean:
	rm -rf ~/rpmbuild/BUILD/${TARGET} ~/rpmbuild/BUILD/${BUILDARCH}/*

install:
	cd ~/rpmbuild/RPMS/${BUILDARCH} && sudo yum install -y `ls -1rt | tail -1`

push:
	git push -u origin $(shell git symbolic-ref --short HEAD)

codebuild:
# Need to run the following first
# aws configure
	aws cloudformation deploy --template-file tools/cfn_codebuild.yml --stack-name trapexBuild --capabilities CAPABILITY_IAM

retry_build:
	aws codebuild retry-build --id trapex
