# https://www.jamescoyle.net/how-to/2448-create-a-simple-cassandra-cluster-with-3-nodes
# https://www.digitalocean.com/community/tutorials/how-to-install-cassandra-and-run-a-single-node-cluster-on-ubuntu-22-04

# Expecting for all:
#CASSANDRA_SEEDS=10.5.0.11,10.5.0.12

# Expecting for each enstance:
#CASSANDRA_IP=10.5.0.11 (12,13)
#CASSANDRA_RACK=rack1 (2,3)

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
sudo sed -i -e "s~endpoint_snitch:[\: \"a-zA-Z0-9\.]*~endpoint_snitch: GossipingPropertyFileSnitch~g" /etc/cassandra/cassandra.yaml

# To avoid "Cannot start node if snitch’s data center (dc1) differs from previous data center (datacenter1)"
# error, keep using dc=datacenter1
sudo sed -i -e "s~rack=[a-zA-Z0-9]*~rack=$CASSANDRA_RACK~g" /etc/cassandra/cassandra-rackdc.properties

sudo rm -f rm /etc/cassandra/cassandra-topology.properties

sudo systemctl start cassandra
if [ "$?" -ne "0" ]; then
    echo Cannot start cassandra, exiting
    exit $?
fi
