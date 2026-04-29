#!/bin/bash

# This part of the script downloads all dependency packages to ../build.
# http://archive.apache.org - very slow
# https://packagecloud.io - usually ok speed, but just in case
# https://github.com - unreliable (DDOS protection?)
# ftp.us.debian.org - usually ok speed, but just in case

# ActiveMQ
curl -Lo ../build/apache-activemq-6.2.5-bin.tar.gz https://archive.apache.org/dist/activemq/6.2.5/apache-activemq-6.2.5-bin.tar.gz
curl -Lo ../build/apache-artemis-2.53.0-bin.tar.gz https://archive.apache.org/dist/artemis/artemis/2.53.0/apache-artemis-2.53.0-bin.tar.gz

# RabbitMQ 4.2.0 and before
# Erlang from cloudamqp 40mb
# curl -Lo ../build/esl-erlang_27.3.4-1_amd64.deb "https://packagecloud.io/cloudamqp/erlang/packages/ubuntu/noble/esl-erlang_27.3.4-1_amd64.deb/download.deb?distro_version_id=284"
# curl -Lo ../build/esl-erlang_27.3.4-1_arm64.deb "https://packagecloud.io/cloudamqp/erlang/packages/ubuntu/noble/esl-erlang_27.3.4-1_arm64.deb/download.deb?distro_version_id=284"
# Rabbitmq 4.2.0 from cloudamqp
# curl -Lo ../build/rabbitmq-server_4.2.0-1_all.deb "https://packagecloud.io/cloudamqp/rabbitmq/packages/any/any/rabbitmq-server_4.2.0-1_all.deb/download.deb?distro_version_id=35"

# RabbitMQ 4.3.0 and after
# Erlang from Ubuntu 11mb (cloudamqp hasn't released it as of Apr 26, 2026)
# https://pkgs.org/download/erlang-base, https://ubuntu.pkgs.org/26.04/ubuntu-main-amd64/erlang-base_27.3.4.6+dfsg-1_amd64.deb.html
curl -Lo ../build/erlang-base_27.3.4.6+dfsg-1_amd64.deb "http://archive.ubuntu.com/ubuntu/pool/main/e/erlang/erlang-base_27.3.4.6+dfsg-1_amd64.deb"
curl -Lo ../build/erlang-base_27.3.4.6+dfsg-1_arm64.deb "http://archive.ubuntu.com/ubuntu/pool/main/e/erlang/erlang-base_27.3.4.6+dfsg-1_arm64.deb"
# RabbitMQ from cloudamqp
 curl -Lo ../build/rabbitmq-server_4.3.0-1_all.deb "https://packagecloud.io/cloudamqp/rabbitmq/packages/any/any/rabbitmq-server_4.3.0-1_all.deb/download.deb?distro_version_id=35"
# rabbitmqadmin from github
curl -Lo ../build/rabbitmqadmin_2.29.0_amd64.deb "https://github.com/rabbitmq/rabbitmqadmin-ng/releases/download/v2.29.0/rabbitmqadmin_2.29.0_amd64.deb"
curl -Lo ../build/rabbitmqadmin_2.29.0_arm64.deb "https://github.com/rabbitmq/rabbitmqadmin-ng/releases/download/v2.29.0/rabbitmqadmin_2.29.0_arm64.deb"

# Prometheus
curl -Lo ../build/prometheus-3.11.3.linux-arm64.tar.gz https://github.com/prometheus/prometheus/releases/download/v3.11.3/prometheus-3.11.3.linux-arm64.tar.gz
curl -Lo ../build/prometheus-3.11.3.linux-amd64.tar.gz https://github.com/prometheus/prometheus/releases/download/v3.11.3/prometheus-3.11.3.linux-amd64.tar.gz
curl -Lo ../build/node_exporter-1.11.1.linux-arm64.tar.gz https://github.com/prometheus/node_exporter/releases/download/v1.11.1/node_exporter-1.11.1.linux-arm64.tar.gz
curl -Lo ../build/node_exporter-1.11.1.linux-amd64.tar.gz https://github.com/prometheus/node_exporter/releases/download/v1.11.1/node_exporter-1.11.1.linux-amd64.tar.gz
curl -Lo ../build/jmx_prometheus_javaagent-1.5.0.jar https://github.com/prometheus/jmx_exporter/releases/download/1.5.0/jmx_prometheus_javaagent-1.5.0.jar

# Cassandra 68mb
curl -Lo ../build/cassandra_5.0.8_all.deb https://apache.jfrog.io/artifactory/cassandra-deb/pool/main/c/cassandra/cassandra_5.0.8_all.deb