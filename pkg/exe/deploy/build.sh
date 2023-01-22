#!/bin/sh
GOOS=linux GOARCH=amd64 go build -o ../../../build/linux/amd64/capidaemon -ldflags="-s -w" ../daemon/capidaemon.go
GOOS=linux GOARCH=amd64 go build -o ../../../build/linux/amd64/capiwebapi -ldflags="-s -w" ../webapi/capiwebapi.go

pushd ../../../ui
set CAPILLARIES_WEBAPI_URL=http://
npm run build
popd
