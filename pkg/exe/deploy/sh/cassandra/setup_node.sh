# https://www.jamescoyle.net/how-to/2448-create-a-simple-cassandra-cluster-with-3-nodes
# https://www.digitalocean.com/community/tutorials/how-to-install-cassandra-and-run-a-single-node-cluster-on-ubuntu-22-04
# https://youkudbhelper.wordpress.com/2020/05/17/cassandradaemon-java731-cannot-start-node-if-snitchs-data-center-dc1-differs-from-previous-data-center-datacenter1/
# https://stackoverflow.com/questions/38961502/cannot-start-cassandra-snitchs-datacenter-differs-from-previous

# Expecting for all:
#CASSANDRA_SEEDS=10.5.0.11,10.5.0.12

# Expecting for each enstance:
#CASSANDRA_IP=10.5.0.11 (12,13)

sudo apt update

echo "deb http://www.apache.org/dist/cassandra/debian 40x main" | sudo tee -a /etc/apt/sources.list.d/cassandra.sources.list

wget -q -O - https://www.apache.org/dist/cassandra/KEYS | sudo tee /etc/apt/trusted.gpg.d/cassandra.asc > /dev/null

sudo apt update

sudo apt -y install cassandra

sudo systemctl status cassandra
if [ "$?" -ne "0" ]; then
    echo Bad cassandra service status, exiting
    exit $?
fi

sudo systemctl stop cassandra

sudo sed -i -e "s~seeds:[\: \"a-zA-Z0-9\.]*~seeds: $CASSANDRA_SEEDS~g" /etc/cassandra/cassandra.yaml
sudo sed -i -e "s~listen_address:[\: \"a-zA-Z0-9\.]*~listen_address: $CASSANDRA_IP~g" /etc/cassandra/cassandra.yaml
sudo sed -i -e "s~rpc_address:[\: \"a-zA-Z0-9\.]*~rpc_address: $CASSANDRA_IP~g" /etc/cassandra/cassandra.yaml
sudo sed -i -e "s~endpoint_snitch:[\: \"a-zA-Z0-9\.]*~endpoint_snitch: SimpleSnitch~g" /etc/cassandra/cassandra.yaml

# We use our test Cassandra setup up to the limit, so avoid "Operation timed out - received only 0 responses"
# https://stackoverflow.com/questions/38231621/cassandra-operation-timed-out
# Test network may be slow, so increase write timeout so coordinater has more time to complete,
# make sure the client (daemon env config cassandra.timeout) has timeout more (not equal) than this, otherwise the client will start throwing
# "no response received from cassandra within timeout period"
# But in general, increasing timeouts from the default 2000 doesn't help much. Improve network/hardware.
sudo sed -i -e "s~write_request_timeout_in_ms:[ ]*[0-9]*~write_request_timeout_in_ms: 10000~g" /etc/cassandra/cassandra.yaml

# Default is 5% of the heap 2000-100mb>, make it bigger (does not help)
# sudo sed -i -e "s~key_cache_size_in_mb:[ 0-9]*~key_cache_size_in_mb: 1000~g" /etc/cassandra/cassandra.yaml
# Do not store keys longer than 120s (does not help)
#sudo sed -i -e "s~key_cache_save_period:[ 0-9]*~key_cache_save_period: 120~g" /etc/cassandra/cassandra.yaml

sudo rm -R /var/lib/cassandra/data

# To avoid "Cannot start node if snitch’s data center (dc1) differs from previous data center (datacenter1)"
# error, keep using dc and rack variables as they are (dc1,rack1) in /etc/cassandra/cassandra-rackdc.properties
# but ignore the dc - it's testing
echo 'JVM_OPTS="$JVM_OPTS -Dcassandra.ignore_dc=true"' | sudo tee -a /etc/cassandra/cassandra-env.sh

# We do not need this config file
sudo rm -f rm /etc/cassandra/cassandra-topology.properties

sudo systemctl start cassandra
if [ "$?" -ne "0" ]; then
    echo Cannot start cassandra, exiting
    exit $?
fi
