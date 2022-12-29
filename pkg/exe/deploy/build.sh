#!/bin/sh
GOOS=linux GOARCH=amd64 go build -o ../../../build/linux/amd64/capidaemon -ldflags="-s -w" ../daemon/daemon.go
/mnt/c/tools/upx.exe ../../../build/linux/amd64/capidaemon
GOOS=linux GOARCH=amd64 go build -o ../../../build/linux/amd64/webapi -ldflags="-s -w" ../webapi/webapi.go
/mnt/c/tools/upx.exe ../../../build/linux/amd64/webapi
