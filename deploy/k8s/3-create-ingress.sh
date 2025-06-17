#!/bin/bash

kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.10.1/deploy/static/provider/cloud/deploy.yaml

#Check
#kubectl get service ingress-nginx-controller --namespace=ingress-nginx

# Use ingress via port-forward
kubectl port-forward service/ui 8080:8080 & kubectl port-forward service/webapi 6543:6543 & kubectl port-forward service/rabbitmq 15672:15672 & kubectl port-forward service/prometheus 9090:9090 &

