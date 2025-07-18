#!/bin/bash

kubectl delete statefulsets,services,deployments -l deployment=capitest
kubectl delete configmap prometheus-config

kubectl get pods