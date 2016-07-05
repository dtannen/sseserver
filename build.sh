#!/bin/bash
export GOPATH=`pwd`
echo "[-] Get Packages"
go get github.com/GeertJohan/go.rice/embedded
go get github.com/azer/debug
go get github.com/garyburd/redigo/redis
go get github.com/daaku/go.zipexe
go get github.com/kardianos/osext
echo "[-] Go Build"
go build -o sseserver
