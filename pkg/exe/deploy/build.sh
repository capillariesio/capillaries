#!/bin/sh
GOOS=linux GOARCH=amd64 go build -o ../../../build/linux/amd64/capidaemon -ldflags="-s -w" ../daemon/daemon.go
