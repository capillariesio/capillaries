#!/bin/bash

# Build images used by test k8s deployment

docker build --no-cache -f ./pkg/exe/daemon/docker/Dockerfile -t daemon .
docker build --no-cache -f ./pkg/exe/webapi/docker/Dockerfile -t webapi .
docker build --no-cache -f ./ui/docker/Dockerfile -t ui .
docker build --no-cache -f ./test/docker/cassandra/Dockerfile -t cassandra .

# To delete local images:
# docker image rm -f daemon webapi ui cassandra
