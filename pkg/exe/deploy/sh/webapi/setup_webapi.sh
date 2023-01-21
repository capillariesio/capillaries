# Expecting
# AMQP_URL=amqp://guest:guest@10.5.0.5/
# CASSANDRA_HOSTS='["10.5.0.11","10.5.0.12","10.5.0.13"]'
# WEBAPI_PORT=6543
# WEBAPI_ACCESS_CONTROL_ACCESS_ORIGIN=http://floating_ip_address
# SFTP_USER=sftpuser

ENV_CONFIG_FILE=/home/ubuntu/bin/env_config.json

sed -i -e 's~"url":[ ]*"[a-zA-Z0-9@\.:\/\-_$ ]*"~"url": "'"$AMQP_URL"'"~g' $ENV_CONFIG_FILE
sed -i -e 's~"hosts\":[ ]*\[[0-9a-zA-Z\.\,\-_ "]*\]~"hosts": '"$CASSANDRA_HOSTS~g" $ENV_CONFIG_FILE
sed -i -e 's~"webapi_port\":[ ]*[0-9]*~"webapi_port": '$WEBAPI_PORT'~g' $ENV_CONFIG_FILE
sed -i -e 's~"access_control_allow_origin\":[ ]*"[0-9a-zA-Z\,\.:\/\-_"]*"~"access_control_allow_origin": "'"$WEBAPI_ACCESS_CONTROL_ACCESS_ORIGIN"'"~g' $ENV_CONFIG_FILE
sed -i -e "s~\"keyspace_replication_config\":[ ]*\"[^\"]*\"~\"keyspace_replication_config\": \"{'class':'SimpleStrategy', 'replication_factor':1}\"~g" $ENV_CONFIG_FILE
sed -i -e "s~\"sftpuser\":[ ]*\"[^\"]*\"~\"sftpuser\": \"/home/ubuntu/.ssh/$SFTP_USER\"~g" $ENV_CONFIG_FILE

sudo rm -fR /var/log/webapi
sudo mkdir /var/log/webapi
sudo chmod 777 /var/log/webapi
sudo chmod 744 /home/ubuntu/bin/webapi

/home/ubuntu/bin/webapi >> /var/log/webapi/webapi.log 2>&1 &



