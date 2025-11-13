#!/bin/bash

# Upload binaries to S3 bucket so they can be used by capillaries-deploy or deploy/tf Terraform script.

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
  echo Also, make sure AWS credentials are in place : AWS_ACCESS_KEY_ID,AWS_SECRET_ACCESS_KEY,AWS_DEFAULT_REGION
  exit 0
fi

echo "Copying in files to "$1
# aws s3 cp ./build/ca $1/ca/ --recursive --include "*"
# aws s3 cp ./build/linux $1/linux/ --recursive --include "*"
# aws s3 cp ./build/webui $1/webui/ --recursive --include "*"

# Ge them from apache, very slow
# curl -Lo ./build/apache-activemq-6.1.8-bin.tar.gz http://archive.apache.org/dist/activemq/6.1.8/apache-activemq-6.1.8-bin.tar.gz
# curl -Lo ./build/apache-artemis-2.44.0-bin.tar.gz https://archive.apache.org/dist/activemq/activemq-artemis/2.44.0/apache-artemis-2.44.0-bin.tar.gz

# Get them from cloudamqp
# curl -Lo ./build/esl-erlang_27.3.4-1_amd64.deb "https://packagecloud.io/cloudamqp/erlang/packages/ubuntu/noble/esl-erlang_27.3.4-1_amd64.deb/download.deb?distro_version_id=284"
# curl -Lo ./build/esl-erlang_27.3.4-1_arm64.deb "https://packagecloud.io/cloudamqp/erlang/packages/ubuntu/noble/esl-erlang_27.3.4-1_arm64.deb/download.deb?distro_version_id=284"
# curl -Lo ./build/rabbitmq-server_4.2.0-1_all.deb "https://packagecloud.io/cloudamqp/rabbitmq/packages/any/any/rabbitmq-server_4.2.0-1_all.deb/download.deb?distro_version_id=35"

# Upload them to our S3 location
# aws s3 cp ./build/apache-activemq-6.1.8-bin.tar.gz $1/
# aws s3 cp ./build/apache-artemis-2.44.0-bin.tar.gz $1/
# aws s3 cp ./build/esl-erlang_27.3.4-1_amd64.deb $1/
# aws s3 cp ./build/esl-erlang_27.3.4-1_arm64.deb $1/
# aws s3 cp ./build/rabbitmq-server_4.2.0-1_all.deb $1/
