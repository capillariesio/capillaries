# https://www.jamescoyle.net/how-to/2448-create-a-simple-cassandra-cluster-with-3-nodes
# https://www.digitalocean.com/community/tutorials/how-to-install-cassandra-and-run-a-single-node-cluster-on-ubuntu-22-04
# https://youkudbhelper.wordpress.com/2020/05/17/cassandradaemon-java731-cannot-start-node-if-snitchs-data-center-dc1-differs-from-previous-data-center-datacenter1/
# https://stackoverflow.com/questions/38961502/cannot-start-cassandra-snitchs-datacenter-differs-from-previous

if [ "$CASSANDRA_SEEDS" = "" ]; then
  echo Error, missing: CASSANDRA_SEEDS=10.5.0.11,10.5.0.12
 exit 1
fi
if [ "$CASSANDRA_IP" = "" ]; then
  echo Error, missing: CASSANDRA_IP=10.5.0.11 or 12 or 13
 exit 1
fi

sudo systemctl stop cassandra

sudo sed -i -e "s~seeds:[\: \"a-zA-Z0-9\.,]*~seeds: $CASSANDRA_SEEDS~g" /etc/cassandra/cassandra.yaml
sudo sed -i -e "s~listen_address:[\: \"a-zA-Z0-9\.]*~listen_address: $CASSANDRA_IP~g" /etc/cassandra/cassandra.yaml
sudo sed -i -e "s~rpc_address:[\: \"a-zA-Z0-9\.]*~rpc_address: $CASSANDRA_IP~g" /etc/cassandra/cassandra.yaml
sudo sed -i -e "s~endpoint_snitch:[\: \"a-zA-Z0-9\.]*~endpoint_snitch: SimpleSnitch~g" /etc/cassandra/cassandra.yaml

# Data on attached volume. Comment out to store data on the ephemeral instance volume at /var/lib/cassandra/data.
#sudo sed -i -e "s~- /var/lib/cassandra/data~- /mnt/data~g" /etc/cassandra/cassandra.yaml
sudo sed -i -e "s~- /var/lib/cassandra/data~- /mnt/ramdisk/data~g" /etc/cassandra/cassandra.yaml

# Commitlog on attached volume. Comment out to store commitlog on the ephemeral instance volume at /var/lib/cassandra/commitlog.
#sudo sed -i -e "s~/var/lib/cassandra/commitlog~/mnt/commitlog~g" /etc/cassandra/cassandra.yaml
sudo sed -i -e "s~/var/lib/cassandra/commitlog~/mnt/ramdisk/commitlog~g" /etc/cassandra/cassandra.yaml

# Minimal number of vnodes, we do not need elasticity
sudo sed -i -e "s~num_tokens:[ 0-9]*~num_tokens: 1~g" /etc/cassandra/cassandra.yaml

# No redundancy
sudo sed -i -e "s~allocate_tokens_for_local_replication_factor: [ 0-9]*~allocate_tokens_for_local_replication_factor: 1~g" /etc/cassandra/cassandra.yaml

# If provided, use initial token list to decrease cluster starting time
if [ "$INITIAL_TOKEN" != "" ]; then
  sudo sed -i -e "s~[ #]*initial_token:[^\n]*~initial_token: $INITIAL_TOKEN~g" /etc/cassandra/cassandra.yaml
fi

# In test env, give enough time to Cassandra coordinator to complete the write (cassandra.yaml write_request_timeout_in_ms)
# so there is no doubt that coordinator is the bottleneck,
# and make sure client time out is more (not equal) than that to avoid gocql error "no response received from cassandra within timeout period".
# In prod environments, increasing write_request_timeout_in_ms and corresponding client timeout is not a solution.
sudo sed -i -e "s~write_request_timeout_in_ms:[ ]*[0-9]*~write_request_timeout_in_ms: 10000~g" /etc/cassandra/cassandra.yaml

# Experimenting with key cache size
# Default is 5% of the heap 2000-100mb>, make it bigger (does not help)
# sudo sed -i -e "s~key_cache_size_in_mb:[ 0-9]*~key_cache_size_in_mb: 1000~g" /etc/cassandra/cassandra.yaml
# Do not store keys longer than 120s (does not help)
#sudo sed -i -e "s~key_cache_save_period:[ 0-9]*~key_cache_save_period: 120~g" /etc/cassandra/cassandra.yaml

sudo rm -fR /var/lib/cassandra/data/*
sudo rm -fR /var/lib/cassandra/commitlog/*

# To avoid "Cannot start node if snitch’s data center (dc1) differs from previous data center (datacenter1)"
# error, keep using dc and rack variables as they are (dc1,rack1) in /etc/cassandra/cassandra-rackdc.properties
# but ignore the dc - it's a testing env
echo 'JVM_OPTS="$JVM_OPTS -Dcassandra.ignore_dc=true"' | sudo tee -a /etc/cassandra/cassandra-env.sh

# Cassandra Prometheus exporter
echo 'JVM_OPTS="$JVM_OPTS -javaagent:/usr/share/cassandra/lib/cassandra-exporter-agent-'${PROMETHEUS_CASSANDRA_EXPORTER_VERSION}'-SNAPSHOT.jar"' | sudo tee -a /etc/cassandra/cassandra-env.sh

# We do not need this config file, delete it
sudo rm -f rm /etc/cassandra/cassandra-topology.properties

sudo systemctl start cassandra
if [ "$?" -ne "0" ]; then
    echo Cannot start cassandra, exiting
    exit $?
fi
