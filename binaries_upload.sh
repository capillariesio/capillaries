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
aws s3 cp ./build/ca $1/ca/ --recursive --include "*"
aws s3 cp ./build/linux $1/linux/ --recursive --include "*"
aws s3 cp ./build/webui $1/webui/ --recursive --include "*"
