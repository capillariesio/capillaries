if [ "$CASSANDRA_HOSTS" = "" ]; then
  echo Error, missing: CASSANDRA_HOSTS='["10.5.0.11","10.5.0.12","10.5.0.13"]'
  exit 1
fi
if [ "$AMQP_URL" = "" ]; then
  echo Error, missing: AMQP_URL=amqp://guest:guest@10.5.0.5/
  exit 1
fi
if [ "$WEBAPI_PORT" = "" ]; then
  echo Error, missing: WEBAPI_PORT=6543
  exit 1
fi
if [ "$SSH_USER" = "" ]; then
  echo Error, missing: SSH_USER=ubuntu
  exit 1
fi

if [ "$EXTERNAL_IP_ADDRESS" = "" ]; then
  echo Error, missing EXTERNAL_IP_ADDRESS=1.2.3.4
  exit 1
fi

WEBAPI_ACCESS_CONTROL_ACCESS_ORIGIN="http://$EXTERNAL_IP_ADDRESS"

echo Configuring Webapi to accept requests only from origin: "$WEBAPI_ACCESS_CONTROL_ACCESS_ORIGIN"

pkill -2 capiwebapi
processid=$(pgrep capiwebapi)
if [ "$processid" != "" ]; then
  pkill -9 capiwebapi
fi

ENV_CONFIG_FILE=/home/$SSH_USER/bin/capiwebapi.json

sed -i -e 's~"url":[ ]*"[a-zA-Z0-9@\.:\/\-_$ ]*"~"url": "'"$AMQP_URL"'"~g' $ENV_CONFIG_FILE
sed -i -e 's~"hosts\":[ ]*\[[0-9a-zA-Z\.\,\-_ "]*\]~"hosts": '"$CASSANDRA_HOSTS~g" $ENV_CONFIG_FILE
sed -i -e 's~"webapi_port\":[ ]*[0-9]*~"webapi_port": '$WEBAPI_PORT'~g' $ENV_CONFIG_FILE
sed -i -e 's~"access_control_allow_origin\":[ ]*"[0-9a-zA-Z\,\.:\/\-_"]*"~"access_control_allow_origin": "'"$WEBAPI_ACCESS_CONTROL_ACCESS_ORIGIN"'"~g' $ENV_CONFIG_FILE
sed -i -e "s~\"keyspace_replication_config\":[ ]*\"[^\"]*\"~\"keyspace_replication_config\": \"{'class':'SimpleStrategy', 'replication_factor':1}\"~g" $ENV_CONFIG_FILE
if [ ! "$SFTP_USER" = "" ]; then
  sed -i -e "s~\"sftpuser\":[ ]*\"[^\"]*\"~\"sftpuser\": \"/home/"$SSH_USER"/.ssh/$SFTP_USER\"~g" $ENV_CONFIG_FILE
fi

sudo rm -fR /var/log/capiwebapi
sudo mkdir /var/log/capiwebapi
sudo chmod 777 /var/log/capiwebapi
sudo chmod 744 /home/$SSH_USER/bin/capiwebapi

/home/$SSH_USER/bin/capiwebapi >> /var/log/capiwebapi/capiwebapi.log 2>&1 &



