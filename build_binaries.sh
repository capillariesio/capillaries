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
# export CAPILLARIES_WEBAPI_URL=http://$EXTERNAL_IP_ADDRESS:6543

pushd ./ui
rm -fR ./build
npm run build
pushd build
tar cvzf webui.tgz *
popd
popd

mv ./ui/build/webui.tgz $DIR_BUILD_WEBUI

echo "Copying certificates to "$DIR_BUILD_CA

cp ./test/ca/*.pem $DIR_BUILD_CA
pushd $DIR_BUILD_CA
tar cvzf ca.tgz *.pem --remove-files
popd

echo "Building "$DIR_BUILD_LINUX_AMD64

GOOS=linux GOARCH=amd64 go build -o $DIR_BUILD_LINUX_AMD64/capidaemon -ldflags="-s -w" ./pkg/exe/daemon/capidaemon.go
cp ./pkg/exe/daemon/capidaemon.json $DIR_BUILD_LINUX_AMD64
gzip -f -k $DIR_BUILD_LINUX_AMD64/capidaemon

GOOS=linux GOARCH=amd64 go build -o $DIR_BUILD_LINUX_AMD64/capiwebapi -ldflags="-s -w" "./pkg/exe/webapi/capiwebapi.go"
cp ./pkg/exe/webapi/capiwebapi.json $DIR_BUILD_LINUX_AMD64
gzip -f -k $DIR_BUILD_LINUX_AMD64/capiwebapi

GOOS=linux GOARCH=amd64 go build -o $DIR_BUILD_LINUX_AMD64/capitoolbelt -ldflags="-s -w" "./pkg/exe/toolbelt/capitoolbelt.go"
cp ./pkg/exe/toolbelt/capitoolbelt.json $DIR_BUILD_LINUX_AMD64
gzip -f -k $DIR_BUILD_LINUX_AMD64/capitoolbelt

GOOS=linux GOARCH=amd64 go build -o $DIR_BUILD_LINUX_AMD64/capiparquet -ldflags="-s -w" ./test/code/parquet/capiparquet.go
gzip -f -k $DIR_BUILD_LINUX_AMD64/capiparquet

echo "Building "$DIR_BUILD_LINUX_ARM64

GOOS=linux GOARCH=arm64 go build -o $DIR_BUILD_LINUX_ARM64/capidaemon -ldflags="-s -w" ./pkg/exe/daemon/capidaemon.go
cp ./pkg/exe/daemon/capidaemon.json $DIR_BUILD_LINUX_ARM64
gzip -f -k $DIR_BUILD_LINUX_ARM64/capidaemon

GOOS=linux GOARCH=arm64 go build -o $DIR_BUILD_LINUX_ARM64/capiwebapi -ldflags="-s -w" ./pkg/exe/webapi/capiwebapi.go
cp ./pkg/exe/webapi/capiwebapi.json $DIR_BUILD_LINUX_ARM64
gzip -f -k $DIR_BUILD_LINUX_ARM64/capiwebapi

GOOS=linux GOARCH=arm64 go build -o $DIR_BUILD_LINUX_ARM64/capitoolbelt -ldflags="-s -w" ./pkg/exe/toolbelt/capitoolbelt.go
cp ./pkg/exe/toolbelt/capitoolbelt.json $DIR_BUILD_LINUX_ARM64
gzip -f -k $DIR_BUILD_LINUX_ARM64/capitoolbelt

GOOS=linux GOARCH=arm64 go build -o $DIR_BUILD_LINUX_ARM64/capiparquet -ldflags="-s -w" ./test/code/parquet/capiparquet.go
gzip -f -k $DIR_BUILD_LINUX_ARM64/capiparquet

# Assuming the bucket $1 (s3://capillaries-release/latest) is publicly accessible for read (no write, and no directory listing)
# {
#     "Version": "2012-10-17",
#     "Statement": [
#         {
#             "Effect": "Allow",
#             "Principal": {
#                 "AWS": "arn:aws:iam::<root_user>:user/capillaries-testuser"
#             },
#             "Action": "s3:ListBucket",
#             "Resource": "arn:aws:s3:::capillaries-release"
#         },
#         {
#             "Effect": "Allow",
#             "Principal": {
#                 "AWS": "arn:aws:iam::<root_user>:user/capillaries-testuser"
#             },
#             "Action": [
#                 "s3:DeleteObject",
#                 "s3:GetObject",
#                 "s3:PutObject"
#             ],
#             "Resource": "arn:aws:s3:::capillaries-release/*"
#         },
#         {
#             "Effect": "Allow",
#             "Principal": "*",
#             "Action": "s3:GetObject",
#             "Resource": "arn:aws:s3:::capillaries-release/*"
#         }
#     ]
# }

if [ "$1" = "" ]; then
  echo No destination S3 url specified, not uploading the binaries
  echo To upload, specify s3 url, for example: s3://capillaries-release/latest
  exit 0
fi

echo "Copying in files to "$1
aws s3 cp ./build/ $1/ --recursive --include "*"
