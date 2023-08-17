
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

# RAM disk size in GB
export RAM_DISK_SIZE=$(awk '/MemFree/ { printf "%.0f\n", $2/1024/2 }' /proc/meminfo)
echo $RAM_DISK_SIZE
sudo mkdir /mnt/ramdisk
sudo chmod 777 /mnt/ramdisk
sudo mount -t tmpfs -o size="$RAM_DISK_SIZE"m myramdisk /mnt/ramdisk
if [ "$?" -ne "0" ]; then
    echo Cannot mount ramdisk, exiting
    exit $?
fi


