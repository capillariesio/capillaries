#!/bin/bash

# Mount 3 Minikube directories that will be used as PVs by Cassandra
sudo mkdir /mnt/local-pv-1
sudo chmod 777 /mnt/local-pv-1
sudo mkdir /mnt/local-pv-2
sudo chmod 777 /mnt/local-pv-2

# Make sure minikube is started: minikube start
minikube ssh -- sudo mkdir /mnt/local-pv-1
minikube ssh -- sudo mkdir /mnt/local-pv-2

kubectl apply -f storage
