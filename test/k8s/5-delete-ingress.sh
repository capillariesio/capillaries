#!/bin/bash

pkill -f forward

kubectl delete all --all -n ingress-nginx
kubectl get all -n ingress-nginx
