#!/bin/sh

cd /usr/local/bin
sed -i -e 's~"url":[ ]*"[a-zA-Z0-9@\.:\/\-_$ ]*"~"url": "'"$AMQP_URL"'"~g' env_config.json
sed -i -e 's~"hosts\":[ ]*\[[0-9a-zA-Z\.\,\-_ "]*\]~"hosts": '"$CASSANDRA_HOSTS~g" env_config.json
sed -i -e 's~"access_control_allow_origin\":[ ]*"[0-9a-zA-Z\,\.:\/\-_"]*"~"access_control_allow_origin": "'"$ACCESS_CONTROL_ACCESS_ORIGIN"'"~g' env_config.json
cat  env_config.json
webapi
