# Expecting
# AMQP_URL=amqp://guest:guest@10.5.0.5/
# CASSANDRA_HOSTS='["10.5.0.11","10.5.0.12","10.5.0.13"]'

ENV_CONFIG_FILE=/home/ubuntu/bin/env_config.json

sed -i -e 's~"url":[ ]*"[a-zA-Z0-9@\.:\/\-_$ ]*"~"url": "'"$AMQP_URL"'"~g' $ENV_CONFIG_FILE
sed -i -e 's~"hosts":[ ]*\[[0-9a-zA-Z\.\,\-_ "]*\]~"hosts": '$CASSANDRA_HOSTS"~g" $ENV_CONFIG_FILE
sed -i -e "s~\"keyspace_replication_config\":[ ]*\"[^\"]*\"~\"keyspace_replication_config\": \"{'class':'SimpleStrategy', 'replication_factor':1}\"~g" $ENV_CONFIG_FILE

sudo rm -fR /var/log/capidaemon
sudo mkdir /var/log/capidaemon
sudo chmod 777 /var/log/capidaemon
sudo chmod 744 /home/ubuntu/bin/capidaemon

/home/ubuntu/bin/capidaemon >> /var/log/capidaemon/capidaemon.log 2>&1 &
