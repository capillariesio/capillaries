#!/bin/bash

# This script uploads dependency packages to our "artifactory" at $1 (s3://capillaries-release/latest)

if [ "$1" = "" ]; then
  echo No destination S3 url specified, not uploading the binaries
  echo To upload, specify s3 url, for example: s3://capillaries-release/latest
  echo Also, make sure AWS credentials are in place : AWS_ACCESS_KEY_ID,AWS_SECRET_ACCESS_KEY,AWS_DEFAULT_REGION
  exit 0
fi

echo "Copying files to "$1

# ActiveMQ
aws s3 cp ../build/apache-activemq-6.2.5-bin.tar.gz $1/
aws s3 cp ../build/apache-artemis-2.53.0-bin.tar.gz $1/

# Erlang from Ubuntu (11mb)
aws s3 cp ../build/erlang-base_27.3.4.6+dfsg-1_amd64.deb $1/
aws s3 cp ../build/erlang-base_27.3.4.6+dfsg-1_arm64.deb $1/

# RabbitMQ server and rabbimqadmin
aws s3 cp ../build/rabbitmq-server_4.3.0-1_all.deb $1/
aws s3 cp ../build/rabbitmqadmin_2.29.0_amd64.deb $1/
aws s3 cp ../build/rabbitmqadmin_2.29.0_arm64.deb $1/

# Prometheus
aws s3 cp ../build/prometheus-3.11.3.linux-arm64.tar.gz $1/
aws s3 cp ../build/prometheus-3.11.3.linux-amd64.tar.gz $1/
aws s3 cp ../build/node_exporter-1.11.1.linux-arm64.tar.gz $1/
aws s3 cp ../build/node_exporter-1.11.1.linux-amd64.tar.gz $1/
aws s3 cp ../build/jmx_prometheus_javaagent-1.5.0.jar $1/

# Cassandra 68mb
aws s3 cp ../build/cassandra_5.0.8_all.deb $1/