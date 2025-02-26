#!/bin/bash

AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
if [ "$?" != "0" ]; then
  echo Cannot obtain AWS account id, make sure AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY are set
  exit 1
fi

if [ "$AWS_DEFAULT_REGION" = "" ]; then
  echo AWS_DEFAULT_REGION is not set
  exit 1
fi

aws ecr get-login-password | docker login --username AWS --password-stdin $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com

# Capillaries daemon and webapi may use S3 storage, provide credentials
kubectl delete secret aws-credentials-secret
kubectl create secret generic aws-credentials-secret --from-literal="AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID" --from-literal="AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY" --from-literal="AWS_DEFAULT_REGION=$AWS_DEFAULT_REGION"

# Allow docker to pull images
kubectl delete secret ecr-pull-secret
kubectl create secret generic ecr-pull-secret --from-file=".dockerconfigjson=/home/$USER/.docker/config.json" --type=kubernetes.io/dockerconfigjson

# Make sure coredns is running: kubectl get pods --namespace=kube-system -l k8s-app=kube-dns
coreDns=$(kubectl get pods --namespace=kube-system -l k8s-app=kube-dns |grep -E 'coredns-[a-z0-9-]+[ ]+1/1[ ]+Running'|wc -l)
if [ "$coreDns" != "1" ]; then
    echo coredns is not running, stopped the rollout
    exit 1;
fi

# kubectl create configmap prometheus-config --from-file=./configmap/prometheus.yaml

kubectl apply -f service

# This is a not-so-elegant way of specifying image names. Consier using Helm/Spinnaker/etc for prod.

replaces="s/{AWS_DEFAULT_REGION}/$AWS_DEFAULT_REGION/; ";
replaces="$replaces s/{AWS_ACCOUNT_ID}/$AWS_ACCOUNT_ID/; ";

# cat ./deployment/daemon.yaml | sed -e "$replaces" | kubectl apply -f -
# cat ./deployment/ui.yaml | sed -e "$replaces" | kubectl apply -f -
# cat ./deployment/webapi.yaml | sed -e "$replaces" | kubectl apply -f -

cat ./statefulset/cassandra.yaml | sed -e "$replaces" | kubectl apply -f -
# kubectl apply -f ./statefulset/prometheus.yaml
# kubectl apply -f ./statefulset/rabbitmq.yaml



