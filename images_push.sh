#!/bin/bash

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

docker tag test_capillaries_containers-webapi $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/webapi
docker tag test_capillaries_containers-daemon $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/daemon
docker tag test_capillaries_containers-ui $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/ui
docker tag test_capillaries_containers-cassandra $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/cassandra

aws ecr get-login-password | docker login --username AWS --password-stdin $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com

docker push $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/webapi
docker push $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/daemon
docker push $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/ui
docker push $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/cassandra

echo Now you can use images
echo $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com/\<webapi,daemin,ui,cassandra\>:latest
