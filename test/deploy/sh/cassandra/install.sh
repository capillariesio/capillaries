
echo "deb https://debian.cassandra.apache.org 41x main" | sudo tee -a /etc/apt/sources.list.d/cassandra.sources.list

curl https://downloads.apache.org/cassandra/KEYS | sudo apt-key add -

sudo apt-get -y update

sudo apt-get install -y cassandra

sudo systemctl status cassandra
if [ "$?" -ne "0" ]; then
    echo Bad cassandra service status, exiting
    exit $?
fi

curl -LO https://github.com/instaclustr/cassandra-exporter/releases/download/v${PROMETHEUS_CASSANDRA_EXPORTER_VERSION}/cassandra-exporter-agent-${PROMETHEUS_CASSANDRA_EXPORTER_VERSION}-SNAPSHOT.jar
if [ "$?" -ne "0" ]; then
    echo Cannot download Prometheus Cassandra exporter, exiting
    exit $?
fi

sudo mv cassandra-exporter-agent-${PROMETHEUS_CASSANDRA_EXPORTER_VERSION}-SNAPSHOT.jar /usr/share/cassandra/lib/