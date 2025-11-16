#!/bin/bash

if [ "$1" = "" ]; then
  echo No destination S3 url specified, not uploading the binaries
  echo To upload, specify s3 url, for example: s3://capillaries-release/latest
  echo Also, make sure AWS credentials are in place : AWS_ACCESS_KEY_ID,AWS_SECRET_ACCESS_KEY,AWS_DEFAULT_REGION
  exit 0
fi

echo "Copying in files to "$1

# Get them from apache, very slow
# curl -Lo ../../../build/apache-activemq-6.1.8-bin.tar.gz http://archive.apache.org/dist/activemq/6.1.8/apache-activemq-6.1.8-bin.tar.gz
# curl -Lo ../../../build/apache-artemis-2.44.0-bin.tar.gz https://archive.apache.org/dist/activemq/activemq-artemis/2.44.0/apache-artemis-2.44.0-bin.tar.gz

# Get them from cloudamqp (speed ok, but just in case)
# curl -Lo ../../../build/esl-erlang_27.3.4-1_amd64.deb "https://packagecloud.io/cloudamqp/erlang/packages/ubuntu/noble/esl-erlang_27.3.4-1_amd64.deb/download.deb?distro_version_id=284"
# curl -Lo ../../../build/esl-erlang_27.3.4-1_arm64.deb "https://packagecloud.io/cloudamqp/erlang/packages/ubuntu/noble/esl-erlang_27.3.4-1_arm64.deb/download.deb?distro_version_id=284"
# curl -Lo ../../../build/rabbitmq-server_4.2.0-1_all.deb "https://packagecloud.io/cloudamqp/rabbitmq/packages/any/any/rabbitmq-server_4.2.0-1_all.deb/download.deb?distro_version_id=35"

# Get them from github, unreliable
# curl -Lo ../../../build/prometheus-3.7.0.linux-arm64.tar.gz https://github.com/prometheus/prometheus/releases/download/v3.7.0/prometheus-3.7.0.linux-arm64.tar.gz
# curl -Lo ../../../build/prometheus-3.7.0.linux-amd64.tar.gz https://github.com/prometheus/prometheus/releases/download/v3.7.0/prometheus-3.7.0.linux-amd64.tar.gz
# curl -Lo ../../../build/node_exporter-1.9.1.linux-arm64.tar.gz https://github.com/prometheus/node_exporter/releases/download/v1.9.1/node_exporter-1.9.1.linux-arm64.tar.gz
# curl -Lo ../../../build/node_exporter-1.9.1.linux-amd64.tar.gz https://github.com/prometheus/node_exporter/releases/download/v1.9.1/node_exporter-1.9.1.linux-amd64.tar.gz
# curl -Lo ../../../build/jmx_prometheus_javaagent-1.5.0.jar https://github.com/prometheus/jmx_exporter/releases/download/1.5.0/jmx_prometheus_javaagent-1.5.0.jar

# Upload them to our S3 location
aws s3 cp ../../../build/apache-activemq-6.1.8-bin.tar.gz $1/
aws s3 cp ../../../build/apache-artemis-2.44.0-bin.tar.gz $1/

aws s3 cp ../../../build/esl-erlang_27.3.4-1_amd64.deb $1/
aws s3 cp ../../../build/esl-erlang_27.3.4-1_arm64.deb $1/
aws s3 cp ../../../build/rabbitmq-server_4.2.0-1_all.deb $1/

aws s3 cp ../../../build/prometheus-3.7.0.linux-arm64.tar.gz $1/
aws s3 cp ../../../build/prometheus-3.7.0.linux-amd64.tar.gz $1/
aws s3 cp ../../../build/node_exporter-1.9.1.linux-arm64.tar.gz $1/
aws s3 cp ../../../build/node_exporter-1.9.1.linux-amd64.tar.gz $1/
aws s3 cp ../../../build/jmx_prometheus_javaagent-1.5.0.jar $1/
