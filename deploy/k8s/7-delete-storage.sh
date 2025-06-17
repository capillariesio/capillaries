#!/bin/bash

# Assuming 2 pvcs were automatically created
kubectl delete pvc cassandra-data-cassandra-0
kubectl delete pvc cassandra-data-cassandra-1

# Delete local pvs created by 0-create-storage.sh
kubectl patch pv local-pv-1 -p '{"spec":{"claimRef": null}}'
kubectl patch pv local-pv-2 -p '{"spec":{"claimRef": null}}'
kubectl delete pv local-pv-1
kubectl delete pv local-pv-2

kubectl delete pvc local-pvc