
BUILDARCH=x86_64
TARGET=trapex

build:
	go build ./...

deps:
	go get ./...

rpm: build
	rpmbuild -ba rpm.spec

clean:
	rm -rf ~/rpmbuild/BUILD/${TARGET} ~/rpmbuild/BUILD/${BUILDARCH}/*

install:
	cd ~/rpmbuild/RPMS/${BUILDARCH} && sudo yum install -y `ls -1rt | tail -1`

push:
	git push -u origin $(shell git symbolic-ref --short HEAD)

