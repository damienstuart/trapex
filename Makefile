
BUILDARCH=x86_64
TARGET=trapex

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

codebuild:
# Need to run the following first
# aws configure
	aws cloudformation deploy --template-file tools/codebuild.yml --stack-name trapexrpm --capabilities CAPABILITY_IAM
	aws cloudformation deploy --template-file tools/codebuild_batch_cfn.yml --stack-name trapexbatchrpm --capabilities CAPABILITY_IAM --parameter-overrides StreamId=rpm BuildSpec=tools/buildspec_batch_rpm.yml
	aws cloudformation deploy --template-file tools/codebuild_batch_cfn.yml --stack-name trapexbatchnopkg --capabilities CAPABILITY_IAM --parameter-overrides StreamId=nopkg BuildSpec=tools/buildspec_batch_nopkg.yml CodeBuildImage=aws/codebuild/standard:5.0 

