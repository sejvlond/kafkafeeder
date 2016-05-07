#!/usr/bin/env bash

set -e

PROJECT=$GOPATH/src/github.com/sejvlond/kafkafeeder

mkdir -p $PROJECT
cd $_
cp -R /src/* .
rm -rf vendor

echo "RUN: glide install"
glide install

echo "RUN: go test ."
go test .

echo "RUN: go build -tags=netgo -o `basename $PROJECT`"
go build -tags=netgo -o `basename $PROJECT`

echo "RUN: dpkg-buildpackage -b -uc"
find . -name Makefile -delete
dpkg-buildpackage -b -uc

echo "mv ../*.deb /src"
mv ../*.deb /src

echo "== ALL DONE =="
