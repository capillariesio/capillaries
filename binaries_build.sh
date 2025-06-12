#!/bin/bash

DIR_BUILD_LINUX_AMD64=./build/linux/amd64
DIR_BUILD_LINUX_ARM64=./build/linux/arm64
DIR_BUILD_CA=./build/ca
DIR_BUILD_WEBUI=./build/webui

rm -fR ./build
mkdir -p $DIR_BUILD_LINUX_AMD64
mkdir -p $DIR_BUILD_LINUX_ARM64
mkdir -p $DIR_BUILD_CA
mkdir -p $DIR_BUILD_WEBUI

echo "Building webui"

# If needed: export PATH=$PATH:$HOME/.nvm/versions/node/v20.9.0/bin
# Just build for http://localhost:6543, sh/webapi/config.sh will patch bundle.js on deployment
# If, for some reason, patching is not an option, build UI with this env variable set:
# export CAPI_WEBAPI_URL=http://$EXTERNAL_IP_ADDRESS:6543

pushd ./ui
rm -fR ./build
npm run build
pushd build
tar cvzf webui.tgz *
popd
popd

mv ./ui/build/webui.tgz $DIR_BUILD_WEBUI

echo "Building "$DIR_BUILD_LINUX_AMD64

GOOS=linux GOARCH=amd64 go build -o $DIR_BUILD_LINUX_AMD64/capidaemon -ldflags="-s -w -X main.version=$(git describe --tags --always)" ./pkg/exe/daemon/capidaemon.go
cp ./pkg/exe/daemon/capidaemon.json $DIR_BUILD_LINUX_AMD64
gzip -f -k $DIR_BUILD_LINUX_AMD64/capidaemon

GOOS=linux GOARCH=amd64 go build -o $DIR_BUILD_LINUX_AMD64/capiwebapi -ldflags="-s -w -X main.version=$(git describe --tags --always)" "./pkg/exe/webapi/capiwebapi.go"
cp ./pkg/exe/webapi/capiwebapi.json $DIR_BUILD_LINUX_AMD64
gzip -f -k $DIR_BUILD_LINUX_AMD64/capiwebapi

GOOS=linux GOARCH=amd64 go build -o $DIR_BUILD_LINUX_AMD64/capitoolbelt -ldflags="-s -w -X main.version=$(git describe --tags --always)" "./pkg/exe/toolbelt/capitoolbelt.go"
cp ./pkg/exe/toolbelt/capitoolbelt.json $DIR_BUILD_LINUX_AMD64
gzip -f -k $DIR_BUILD_LINUX_AMD64/capitoolbelt

GOOS=linux GOARCH=amd64 go build -o $DIR_BUILD_LINUX_AMD64/capiparquet -ldflags="-s -w -X main.version=$(git describe --tags --always)" ./test/code/parquet/capiparquet.go
gzip -f -k $DIR_BUILD_LINUX_AMD64/capiparquet

echo "Building "$DIR_BUILD_LINUX_ARM64

GOOS=linux GOARCH=arm64 go build -o $DIR_BUILD_LINUX_ARM64/capidaemon -ldflags="-s -w -X main.version=$(git describe --tags --always)" ./pkg/exe/daemon/capidaemon.go
cp ./pkg/exe/daemon/capidaemon.json $DIR_BUILD_LINUX_ARM64
gzip -f -k $DIR_BUILD_LINUX_ARM64/capidaemon

GOOS=linux GOARCH=arm64 go build -o $DIR_BUILD_LINUX_ARM64/capiwebapi -ldflags="-s -w -X main.version=$(git describe --tags --always)" ./pkg/exe/webapi/capiwebapi.go
cp ./pkg/exe/webapi/capiwebapi.json $DIR_BUILD_LINUX_ARM64
gzip -f -k $DIR_BUILD_LINUX_ARM64/capiwebapi

GOOS=linux GOARCH=arm64 go build -o $DIR_BUILD_LINUX_ARM64/capitoolbelt -ldflags="-s -w -X main.version=$(git describe --tags --always)" ./pkg/exe/toolbelt/capitoolbelt.go
cp ./pkg/exe/toolbelt/capitoolbelt.json $DIR_BUILD_LINUX_ARM64
gzip -f -k $DIR_BUILD_LINUX_ARM64/capitoolbelt

GOOS=linux GOARCH=arm64 go build -o $DIR_BUILD_LINUX_ARM64/capiparquet -ldflags="-s -w -X main.version=$(git describe --tags --always)" ./test/code/parquet/capiparquet.go
gzip -f -k $DIR_BUILD_LINUX_ARM64/capiparquet
