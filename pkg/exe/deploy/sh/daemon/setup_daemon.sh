# Expecting
# AMQP_URL=amqp://guest:guest@10.5.0.5/
# CASSANDRA_HOSTS='["10.5.0.11","10.5.0.12","10.5.0.13"]'
# SFTP_USER=sftpuser

ENV_CONFIG_FILE=/home/ubuntu/bin/env_config.json

sed -i -e 's~"url":[ ]*"[a-zA-Z0-9@\.:\/\-_$ ]*"~"url": "'"$AMQP_URL"'"~g' $ENV_CONFIG_FILE
sed -i -e 's~"hosts":[ ]*\[[0-9a-zA-Z\.\,\-_ "]*\]~"hosts": '$CASSANDRA_HOSTS"~g" $ENV_CONFIG_FILE
# We use our test Cassandra setup up to the limit, so avoid "Operation timed out - received only 0 responses"
# https://stackoverflow.com/questions/38231621/cassandra-operation-timed-out
# Make replication factor at least 2 to make reads more available, 1 for faster writes
sed -i -e "s~\"keyspace_replication_config\":[ ]*\"[^\"]*\"~\"keyspace_replication_config\": \"{'class':'SimpleStrategy', 'replication_factor':2}\"~g" $ENV_CONFIG_FILE

# In test env, give enough time to Cassandra coordinator to complete the write (cassandra.yaml write_request_timeout_in_ms)
# and make sure client time out is more (not equal) than that
sed -i -e "s~\"timeout\":[ ]*[0-9]*~\"timeout\": 15000~g" $ENV_CONFIG_FILE

# Default value of 50 writer workers is pretty aggressive and can bring test Cassandra cluster to its knees, try 20
sed -i -e "s~\"writer_workers\":[ ]*[0-9]*~\"writer_workers\": 20~g" $ENV_CONFIG_FILE

sed -i -e "s~\"sftpuser\":[ ]*\"[^\"]*\"~\"sftpuser\": \"/home/ubuntu/.ssh/$SFTP_USER\"~g" $ENV_CONFIG_FILE

sudo rm -fR /var/log/capidaemon
sudo mkdir /var/log/capidaemon
sudo chmod 777 /var/log/capidaemon
sudo chmod 744 /home/ubuntu/bin/capidaemon

/home/ubuntu/bin/capidaemon >> /var/log/capidaemon/capidaemon.log 2>&1 &

