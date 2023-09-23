#!/bin/bash
if [ "$DIR_BUILD_LINUX_AMD64" = "" ]; then
  echo Error, missing DIR_BUILD_LINUX_AMD64=../../../../build/linux/amd64
  exit 1
fi
if [ "$DIR_BUILD_LINUX_ARM64" = "" ]; then
  echo Error, missing DIR_BUILD_LINUX_ARM64=../../../../build/linux/arm64
  exit 1
fi
if [ "$DIR_PKG_EXE" = "" ]; then
  echo Error, missing DIR_PKG_EXE=../../../../pkg/exe
  exit 1
fi
if [ "$DIR_CODE_PARQUET" = "" ]; then
  echo Error, missing DIR_CODE_PARQUET=../../../code/parquet
  exit 1
fi

# Assuming HOME is set by ExecLocal
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export GOCACHE="$HOME/.cache/go-build"
export GOMODCACHE="$HOME/go/pkg/mod"

if [ ! -d $DIR_BUILD_LINUX_AMD64 ]; then
  mkdir -p $DIR_BUILD_LINUX_AMD64
fi

if [ ! -d $DIR_BUILD_LINUX_ARM64 ]; then
  mkdir -p $DIR_BUILD_LINUX_ARM64
fi

GOOS=linux GOARCH=amd64 go build -o $DIR_BUILD_LINUX_AMD64/capidaemon -ldflags="-s -w" $DIR_PKG_EXE/daemon/capidaemon.go
gzip -f $DIR_BUILD_LINUX_AMD64/capidaemon
GOOS=linux GOARCH=amd64 go build -o $DIR_BUILD_LINUX_AMD64/capiwebapi -ldflags="-s -w" $DIR_PKG_EXE/webapi/capiwebapi.go
gzip -f $DIR_BUILD_LINUX_AMD64/capiwebapi
GOOS=linux GOARCH=amd64 go build -o $DIR_BUILD_LINUX_AMD64/capitoolbelt -ldflags="-s -w" $DIR_PKG_EXE/toolbelt/capitoolbelt.go
gzip -f $DIR_BUILD_LINUX_AMD64/capitoolbelt
GOOS=linux GOARCH=amd64 go build -o $DIR_BUILD_LINUX_AMD64/capiparquet -ldflags="-s -w" $DIR_CODE_PARQUET/capiparquet.go
gzip -f $DIR_BUILD_LINUX_AMD64/capiparquet

GOOS=linux GOARCH=arm64 go build -o $DIR_BUILD_LINUX_ARM64/capidaemon -ldflags="-s -w" $DIR_PKG_EXE/daemon/capidaemon.go
gzip -f $DIR_BUILD_LINUX_ARM64/capidaemon
GOOS=linux GOARCH=arm64 go build -o $DIR_BUILD_LINUX_ARM64/capiwebapi -ldflags="-s -w" $DIR_PKG_EXE/webapi/capiwebapi.go
gzip -f $DIR_BUILD_LINUX_ARM64/capiwebapi
GOOS=linux GOARCH=arm64 go build -o $DIR_BUILD_LINUX_ARM64/capitoolbelt -ldflags="-s -w" $DIR_PKG_EXE/toolbelt/capitoolbelt.go
gzip -f $DIR_BUILD_LINUX_ARM64/capitoolbelt
GOOS=linux GOARCH=arm64 go build -o $DIR_BUILD_LINUX_ARM64/capiparquet -ldflags="-s -w" $DIR_CODE_PARQUET/capiparquet.go
gzip -f $DIR_BUILD_LINUX_ARM64/capiparquet
