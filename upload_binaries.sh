#!/bin/bash

if [ "$1" = "" ]; then
  echo No destination S3 url specified, not uploading the binaries
  echo To upload, specify s3 url, for example: s3://capillaries-release/latest
  exit 0
fi

echo "Copying in files to "$1
aws s3 cp ./build/ $1/ --recursive --include "*"
