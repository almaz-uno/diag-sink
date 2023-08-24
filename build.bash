#!/bin/bash

cd $(dirname $(realpath $0))

EXE=diag-sink-$(git describe --tags --abbrev=0)

mkdir -p .build

GOOS=linux GOARCH=amd64 go build -o .build/$EXE.linux
GOOS=darwin GOARCH=amd64 go build -o .build/$EXE.darwin
GOOS=windows GOARCH=amd64 go build -o .build/$EXE.exe
