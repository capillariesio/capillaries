#!/bin/bash

# Push images used by test k8s deployment

AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
if [ "$?" != "0" ]; then
  echo Cannot obtain AWS account id, make sure AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY are set
  exit 1
fi

if [ "$AWS_DEFAULT_REGION" = "" ]; then
  echo AWS_DEFAULT_REGION is set
  exit 1
fi

aws ecr describe-repositories --repository-names daemon || aws ecr create-repository --repository-name daemon
aws ecr describe-repositories --repository-names webapi || aws ecr create-repository --repository-name webapi
aws ecr describe-repositories --repository-names ui || aws ecr create-repository --repository-name ui
aws ecr describe-repositories --repository-names cassandra || aws ecr create-repository --repository-name cassandra

docker tag daemon $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/daemon
docker tag webapi $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/webapi
docker tag ui $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/ui
docker tag cassandra $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/cassandra

aws ecr get-login-password | docker login --username AWS --password-stdin $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com

docker push $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/daemon
docker push $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/webapi
docker push $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/ui
docker push $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/cassandra

# Delete local tagged copies
docker image rm -f $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/daemon
docker image rm -f $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/webapi
docker image rm -f $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/ui
docker image rm -f $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/cassandra

echo Now you can use images
echo $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/\<webapi,daemon,ui,cassandra\>:latest
