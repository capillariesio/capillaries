#!/bin/sh

cd /usr/local/bin
sed -i -e 's~"url":[ ]*"[a-zA-Z0-9@\.:\/\-_$ ]*"~"url": "'"$AMQP_URL"'"~g' capidaemon.json
sed -i -e 's~"hosts":[ ]*\[[0-9a-zA-Z\.\,\-_ "]*\]~"hosts": '"$CASSANDRA_HOSTS~g" capidaemon.json
sed -i -e 's~"ca_path":[ ]*\"[0-9a-zA-Z\.\,\-_\/ ]*\"~"ca_path": "/usr/src/capillaries/test/ca"~g' capidaemon.json
cat  capidaemon.json
capidaemon >> /tmp/capi_out/capidaemon-$(date +"%Y%m%d%H%M").log 2>&1

