#!/bin/bash

echo Running cassandra.sh in $(pwd) ...

if [ "$SSH_USER" = "" ]; then
  echo Error, missing: SSH_USER=ubuntu
  exit 1
fi
if [ "$S3_LOG_URL" = "" ]; then
  echo Error, missing: S3_LOG_URL=s3://capillaries-testbucket/log
  exit 1
fi
if [ "$PROMETHEUS_JMX_EXPORTER_FILENAME" = "" ]; then
  echo Error, missing: PROMETHEUS_JMX_EXPORTER_FILENAME=jmx_prometheus_javaagent-1.5.0.jar
  exit 1
fi
if [ "$PROMETHEUS_NODE_EXPORTER_FILENAME" = "" ]; then
  echo Error, missing: PROMETHEUS_NODE_EXPORTER_FILENAME=node_exporter-1.9.1.linux-amd64.tar.gz
  exit 1
fi
if [ "$CASSANDRA_VERSION" = "" ]; then
  echo Error, missing: CASSANDRA_VERSION=50x
  exit 1
fi
if [ "$CASSANDRA_HOSTS" = "" ]; then
  echo Error, missing: CASSANDRA_HOSTS=10.5.0.11,10.5.0.12
  exit 1
fi
if [ "$CAPILLARIES_RELEASE_URL" = "" ]; then
  echo Error, missing: CAPILLARIES_RELEASE_URL=https://capillaries-release.s3.us-east-1.amazonaws.com/latest
  exit 1
fi
if [ "$CASSANDRA_INTERNAL_IP" = "" ]; then
  echo Error, missing: CASSANDRA_INTERNAL_IP=10.5.0.11
  exit 1
fi
if [ "$CASSANDRA_INITIAL_TOKEN" = "" ]; then
  echo Error, missing: CASSANDRA_INITIAL_TOKEN=0
  exit 1
fi
if [ "$CASSANDRA_NVME_REGEX" = "" ]; then
  echo Error, missing: CASSANDRA_NVME_REGEX="nvme[0-9]n[0-9] 139.7G"
  exit 1
fi

# Use $SSH_USER
if [ ! -d /home/$SSH_USER ]; then
  mkdir -p /home/$SSH_USER
fi
sudo chmod 755 /home/$SSH_USER




# Download and config node exporter, this section is common for all instances


sudo useradd --no-create-home --shell /bin/false node_exporter
cd /home/$SSH_USER
curl -LOs $CAPILLARIES_RELEASE_URL/$PROMETHEUS_NODE_EXPORTER_FILENAME
if [ "$?" -ne "0" ]; then
    echo Cannot download, exiting
    exit $?
fi
tar xvf $PROMETHEUS_NODE_EXPORTER_FILENAME
PROMETHEUS_NODE_EXPORTER_DIR=$(basename $PROMETHEUS_NODE_EXPORTER_FILENAME .tar.gz)
sudo cp $PROMETHEUS_NODE_EXPORTER_DIR/node_exporter /usr/local/bin
sudo chown node_exporter:node_exporter /usr/local/bin/node_exporter
rm -fR $PROMETHEUS_NODE_EXPORTER_FILENAME $PROMETHEUS_NODE_EXPORTER_DIR
PROMETHEUS_NODE_EXPORTER_SERVICE_FILE=/etc/systemd/system/node_exporter.service
sudo rm -f $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE
sudo tee $PROMETHEUS_NODE_EXPORTER_SERVICE_FILE <<EOF
[Unit]
Description=Prometheus Node Exporter
Wants=network-online.target
After=network-online.target
[Service]
User=node_exporter
Group=node_exporter
Type=simple
ExecStart=/usr/local/bin/node_exporter
[Install]
WantedBy=multi-user.target
EOF
sudo systemctl daemon-reload
sudo systemctl start node_exporter
sudo systemctl status node_exporter




# Prepare to install Cassandra




echo "deb https://debian.cassandra.apache.org $CASSANDRA_VERSION main" | sudo tee -a /etc/apt/sources.list.d/cassandra.sources.list
# apt-key is deprecated. but still working, just silence it
curl -s https://downloads.apache.org/cassandra/KEYS | sudo apt-key add - 2>/dev/null

# To avoid "Key is stored in legacy trusted.gpg keyring" in stderr
cd /etc/apt
sudo cp trusted.gpg trusted.gpg.d

sudo DEBIAN_FRONTEND=noninteractive apt-get update -y




# Install iostat for troubleshooting



sudo DEBIAN_FRONTEND=noninteractive apt-get install -y sysstat
if [ "$?" -ne "0" ]; then
    echo sysstat install error, exiting
    exit $?
fi




# Install Java, Cassandra requires Java 8



# Anything above 8 gives "the security manager is deprecated and will be removed in a future release" error
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y openjdk-8-jdk
if [ "$?" -ne "0" ]; then
    echo openjdk install error, exiting
    exit $?
fi




# Install Cassandra




sudo DEBIAN_FRONTEND=noninteractive apt-get install -y cassandra
if [ "$?" -ne "0" ]; then
    echo cassandra install error, exiting
    exit $?
fi

sudo systemctl status cassandra
if [ "$?" -ne "0" ]; then
    echo Bad cassandra service status, exiting
    exit $?
fi




# Install JMX Exporter




cd /home/$SSH_USER
curl -LOs $CAPILLARIES_RELEASE_URL/$PROMETHEUS_JMX_EXPORTER_FILENAME
if [ "$?" -ne "0" ]; then
    echo Cannot download JMX exporter, exiting
    exit $?
fi
sudo mv $PROMETHEUS_JMX_EXPORTER_FILENAME /usr/share/cassandra/lib/
sudo chown cassandra /usr/share/cassandra/lib/$PROMETHEUS_JMX_EXPORTER_FILENAME
cat > jmx_exporter.yml << 'endmsgmarker'
lowercaseOutputLabelNames: true
lowercaseOutputName: true
whitelistObjectNames: ["org.apache.cassandra.metrics:*"]
# ColumnFamily is an alias for Table metrics
blacklistObjectNames: ["org.apache.cassandra.metrics:type=ColumnFamily,*"]
rules:
# Generic gauges with 0-2 labels
- pattern: org.apache.cassandra.metrics<type=(\S*)(?:, ((?!scope)\S*)=(\S*))?(?:, scope=(\S*))?, name=(\S*)><>Value
  name: cassandra_$1_$5
  type: GAUGE
  labels:
    "$1": "$4"
    "$2": "$3"
#
# Emulate Prometheus 'Summary' metrics for the exported 'Histogram's.
# TotalLatency is the sum of all latencies since server start
#
- pattern: org.apache.cassandra.metrics<type=(\S*)(?:, ((?!scope)\S*)=(\S*))?(?:, scope=(\S*))?, name=(.+)?(?:Total)(Latency)><>Count
  name: cassandra_$1_$5$6_seconds_sum
  type: UNTYPED
  labels:
    "$1": "$4"
    "$2": "$3"
  # Convert microseconds to seconds
  valueFactor: 0.000001
- pattern: org.apache.cassandra.metrics<type=(\S*)(?:, ((?!scope)\S*)=(\S*))?(?:, scope=(\S*))?, name=((?:.+)?(?:Latency))><>Count
  name: cassandra_$1_$5_seconds_count
  type: UNTYPED
  labels:
    "$1": "$4"
    "$2": "$3"
- pattern: org.apache.cassandra.metrics<type=(\S*)(?:, ((?!scope)\S*)=(\S*))?(?:, scope=(\S*))?, name=(.+)><>Count
  name: cassandra_$1_$5_count
  type: UNTYPED
  labels:
    "$1": "$4"
    "$2": "$3"
- pattern: org.apache.cassandra.metrics<type=(\S*)(?:, ((?!scope)\S*)=(\S*))?(?:, scope=(\S*))?, name=((?:.+)?(?:Latency))><>(\d+)thPercentile
  name: cassandra_$1_$5_seconds
  type: GAUGE
  labels:
    "$1": "$4"
    "$2": "$3"
    quantile: "0.$6"
  # Convert microseconds to seconds
  valueFactor: 0.000001
- pattern: org.apache.cassandra.metrics<type=(\S*)(?:, ((?!scope)\S*)=(\S*))?(?:, scope=(\S*))?, name=(.+)><>(\d+)thPercentile
  name: cassandra_$1_$5
  type: GAUGE
  labels:
    "$1": "$4"
    "$2": "$3"
    quantile: "0.$6"
endmsgmarker
sudo mv jmx_exporter.yml /etc/cassandra/
sudo chown cassandra /etc/cassandra/jmx_exporter.yml



# Disable security mgr to avoid "The Security Manager is deprecated and will be removed in a future release" for Java above 8
# echo 'JVM_OPTS="$JVM_OPTS -J-DTopSecurityManager.disable=true"' | sudo tee -a /etc/cassandra/cassandra-env.sh

# Let Cassandra know about JMX Exporter and config
echo 'JVM_OPTS="$JVM_OPTS -javaagent:/usr/share/cassandra/lib/'$PROMETHEUS_JMX_EXPORTER_FILENAME'=7070:/etc/cassandra/jmx_exporter.yml"' | sudo tee -a /etc/cassandra/cassandra-env.sh





# Stop Cassandra for reconfiguration




echo Stopping Cassandra after installation...
sudo systemctl stop cassandra




# Reconfigure Cassandra




# Cassandra 5.0 has a habit to make it drwxr-x---. Make it drwxr-xr-x
echo Changing /var/log/cassandra permissions...
sudo chmod 755 /var/log/cassandra

# Cluster stuff
sudo sed -i -e "s~seeds:[\: \"a-zA-Z0-9\.,]*~seeds: $CASSANDRA_HOSTS~g" /etc/cassandra/cassandra.yaml
sudo sed -i -e "s~listen_address:[\: \"a-zA-Z0-9\.]*~listen_address: $CASSANDRA_INTERNAL_IP~g" /etc/cassandra/cassandra.yaml
sudo sed -i -e "s~rpc_address:[\: \"a-zA-Z0-9\.]*~rpc_address: $CASSANDRA_INTERNAL_IP~g" /etc/cassandra/cassandra.yaml
sudo sed -i -e "s~endpoint_snitch:[\: \"a-zA-Z0-9\.]*~endpoint_snitch: SimpleSnitch~g" /etc/cassandra/cassandra.yaml

# Rotate debug.log every minute, and compress it with gzip, not zip
sudo sed -i -e "s~debug.log.%d{yyyy-MM-dd}.%i.zip~debug.log.%d{\"yyyy-MM-dd'T'HH-mm\"}.%i.gz~g" /etc/cassandra/logback.xml
# Keep at least 60 min of debug.log and system.log history
sudo sed -i -e "s~<maxHistory>7</maxHistory>~<maxHistory>60</maxHistory>~g" /etc/cassandra/logback.xml
# Add node address to log msg
sudo sed -i -e "s~<pattern>%-5level~<pattern>$HOSTNAME %-5level~g" /etc/cassandra/logback.xml

# No need to logrotate, Cassandra uses log4j, configure it conservatively
sudo sed -i -e "s~<maxFileSize>[^<]*</maxFileSize>~<maxFileSize>10MB</maxFileSize>~g" /etc/cassandra/logback.xml
sudo sed -i -e "s~<totalSizeCap>[^<]*</totalSizeCap>~<totalSizeCap>1GB</totalSizeCap>~g" /etc/cassandra/logback.xml

# Data on attached volume
sudo sed -i -e "s~- /var/lib/cassandra/data~~g" /etc/cassandra/cassandra.yaml
# One disk or two disks (Cassandra instances can have one ore two nvme drives)
if [ -d "/data1" ]; then
  sudo sed -i -e "s~data_file_directories:[^\n]*~data_file_directories: [ /data0/d, /data1/d ]~g" /etc/cassandra/cassandra.yaml
else 
  sudo sed -i -e "s~data_file_directories:[^\n]*~data_file_directories: [ /data0/d ]~g" /etc/cassandra/cassandra.yaml
fi

# Commitlog on attached volume
sudo sed -i -e "s~commitlog_directory:[^\n]*~commitlog_directory: /data0/c~g" /etc/cassandra/cassandra.yaml

# Minimal number of vnodes, we do not need elasticity
sudo sed -i -e "s~num_tokens:[ 0-9]*~num_tokens: 1~g" /etc/cassandra/cassandra.yaml

# No redundancy
sudo sed -i -e "s~allocate_tokens_for_local_replication_factor: [ 0-9]*~allocate_tokens_for_local_replication_factor: 1~g" /etc/cassandra/cassandra.yaml

# If provided, use initial token list to decrease cluster starting time
if [ "$CASSANDRA_INITIAL_TOKEN" != "" ]; then
  sudo sed -i -e "s~[ #]*initial_token:[^\n]*~initial_token: $CASSANDRA_INITIAL_TOKEN~g" /etc/cassandra/cassandra.yaml
fi

# In test env, give enough time to Cassandra coordinator to complete the write (cassandra.yaml write_request_timeout_in_ms)
# so there is no doubt that coordinator is the bottleneck,
# and make sure client time out is more (not equal) than that to avoid gocql error "no response received from cassandra within timeout period".
# In prod environments, increasing write_request_timeout_in_ms and corresponding client timeout is not a solution.
sudo sed -i -e "s~write_request_timeout_in_ms:[ ]*[0-9]*~write_request_timeout_in_ms: 10000~g" /etc/cassandra/cassandra.yaml

# Complete data reset
sudo rm -fR /var/lib/cassandra/data/*
sudo rm -fR /var/lib/cassandra/commitlog/*
if [ ! -d "/data0" ]; then
  sudo rm -fR /data0/*
fi
if [ ! -d "/data1" ]; then
  sudo rm -fR /data1/*
fi
sudo rm -fR /var/lib/cassandra/saved_caches/*

# To avoid "Cannot start node if snitch’s data center (dc1) differs from previous data center (datacenter1)"
# error, keep using dc and rack variables as they are (dc1,rack1) in /etc/cassandra/cassandra-rackdc.properties
# but ignore the dc - it's a testing env
echo 'JVM_OPTS="$JVM_OPTS -Dcassandra.ignore_dc=true"' | sudo tee -a /etc/cassandra/cassandra-env.sh

# We do not need this config file, delete it
sudo rm -f rm /etc/cassandra/cassandra-topology.properties

# Helper function
mount_device(){
	local mount_dir="/data"$1
 	local device_name=$2
    echo Checking to mount $device_name at $mount_dir ...
    if [ "$(lsblk -f | grep -E $device_name'[ ]+xfs')" == "" ]; then
      echo Formatting partition ...
	    sudo mkfs -t xfs /dev/$device_name
    else
      echo Partition already formatted
    fi
    if [ ! -d "$mount_dir" ]; then
      echo Creating $mount_dir ...
	  sudo mkdir $mount_dir
    else
      echo $mount_dir already created
    fi
    if [ "$(lsblk -f | grep $mount_dir)" == "" ]; then
      echo Mounting /dev/$device_name as $mount_dir ... 
	    sudo mount /dev/$device_name $mount_dir
    else
      echo Already mounted
    fi
	sudo chown cassandra $mount_dir
	sudo chmod 777 $mount_dir;
}

# "nvme[0-9]n[0-9] 558.8G"
# "loop[0-9] [0-9.]+M"
device_number=0
lsblk | awk '{print $1,$4}' | grep -E "$CASSANDRA_NVME_REGEX" | awk '{print $1}' |
while read -r device_name; do
  mount_device $device_number $device_name
  device_number=$((device_number+1)) 
done



# S3 log location



echo Checking access to $S3_LOG_URL...
aws s3 ls $S3_LOG_URL/

# Add hostname to the log file names and send them to S3 every 5 min
SEND_LOGS_FILE=/home/$SSH_USER/sendlogs.sh
sudo tee $SEND_LOGS_FILE <<EOF
#!/bin/bash
for f in /var/log/cassandra/debug.log.*.gz; do
  if [ -e \$f ]; then
    # Cassandra produces: debug.log.2025-06-12T18-42.3.gz
    # Add hostname to it and rearrange name components: cassandra-2025-06-12T18-42-03.000.ip-10-5-0-11.log.gz
    fname=\$(basename -- "\$f")
    fnamedatetime=\$(echo \$fname| cut -d'.' -f3)
    fnameidx=\$(echo \$fname| cut -d'.' -f4)
    newfilepath=/var/log/cassandra/cassandra-\$fnamedatetime-0\$fnameidx.000.\$HOSTNAME.log.gz
    filesize=\$(stat --print="%s" \$f)
    # Do not send empty files
    if [[ \$filesize -ge 200 ]]; then
       echo \$newfilepath not empty \$filesize bytes
       sudo mv \$f \$newfilepath
       aws s3 cp \$newfilepath $S3_LOG_URL/
       sudo rm \$newfilepath
    else
       echo \$newfilepath empty \$filesize bytes
    fi
  fi
done
EOF
sudo chmod 744 $SEND_LOGS_FILE
sudo su $SSH_USER -c "echo \"*/5 * * * * $SEND_LOGS_FILE\" | crontab -"


# Everything in ~ should belong to ssh user
sudo chown -R $SSH_USER /home/$SSH_USER


# Check node exporter, by now it should be up
curl -s http://localhost:9100/metrics >/dev/null
if [ "$?" -ne "0" ]; then
    echo localhost:9100/metrics failed with $?
fi


# Start Cassandra after reconfiguring it




sudo systemctl start cassandra
if [ "$?" -ne "0" ]; then
    echo Cannot start cassandra, exiting
    exit $?
fi
