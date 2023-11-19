#!/bin/sh

cd /usr/local/bin
sed -i -e 's~"url":[ ]*"[a-zA-Z0-9@\.:\/\-_$ ]*"~"url": "'"$AMQP_URL"'"~g' capi*.json
sed -i -e 's~"hosts\":[ ]*\[[0-9a-zA-Z\.\,\-_ "]*\]~"hosts": '"$CASSANDRA_HOSTS~g" capi*.json
sed -i -e 's~"ca_path":[ ]*\"[0-9a-zA-Z\.\,\-_\/ ]*\"~"ca_path": "/usr/src/capillaries/test/ca"~g' capi*.json
sed -i -e 's~"webapi_port\":[ ]*[0-9]*~"webapi_port": '$WEBAPI_PORT'~g' capiwebapi.json
sed -i -e 's~"access_control_allow_origin\":[ ]*"[0-9a-zA-Z\,\.:\/\-_"]*"~"access_control_allow_origin": "'"$ACCESS_CONTROL_ACCESS_ORIGIN"'"~g' capiwebapi.json
cat  capiwebapi.json
cat  capitoolbelt.json
capiwebapi >> /tmp/capi_out/capiwebapi-$(date +"%Y%m%d%H%M").log 2>&1