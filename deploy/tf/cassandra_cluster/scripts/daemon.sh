#!/bin/bash

echo Running daemon.sh in $(pwd)

if [ "$DAEMON_GOMEMLIMIT_GB" = "" ]; then
  echo Error, missing: DAEMON_GOMEMLIMIT_GB=2
  exit 1
fi
if [ "$DAEMON_GOGC" = "" ]; then
  echo Error, missing: DAEMON_GOGC=100
  exit 1
fi
if [ "$SSH_USER" = "" ]; then
  echo Error, missing: SSH_USER=ubuntu
  exit 1
fi
if [ "$AWSREGION" = "" ]; then
  echo Error, missing: AWSREGION=us-east-1
  exit 1
fi
if [ "$CAPILLARIES_RELEASE_URL" = "" ]; then
  echo Error, missing: CAPILLARIES_RELEASE_URL=https://capillaries-release.s3.us-east-1.amazonaws.com/latest
  exit 1
fi
if [ "$OS_ARCH" = "" ]; then
  echo Error, missing: OS_ARCH=linux/arm64
  exit 1
fi
if [ "$S3_LOG_URL" = "" ]; then
  echo Error, missing: S3_LOG_URL=s3://capillaries-testbucket/log
  exit 1
fi
if [ "$CASSANDRA_HOSTS" = "" ]; then
  echo Error, missing: CASSANDRA_HOSTS=10.5.0.11,10.5.0.12
  exit 1
fi
if [ "$CASSANDRA_PORT" = "" ]; then
  echo Error, missing: CASSANDRA_PORT=9042
  exit 1
fi
if [ "$CASSANDRA_USERNAME" = "" ]; then
  echo Error, missing: CASSANDRA_USERNAME=cassandra
  exit 1
fi
if [ "$CASSANDRA_PASSWORD" = "" ]; then
  echo Error, missing: CASSANDRA_PASSWORD=cassandra
  exit 1
fi
if [ "$RABBITMQ_URL" = "" ]; then
  echo Error, missing: RABBITMQ_URL=amqps://capiuser:capipass@10.5.1.10/
  exit 1
fi
if [ "$PROMETHEUS_NODE_EXPORTER_VERSION" = "" ]; then
  echo Error, missing: PROMETHEUS_NODE_EXPORTER_VERSION=1.2.3
  exit 1
fi
if [ "$WRITER_WORKERS" = "" ]; then
  echo Error, missing: WRITER_WORKERS=4
  exit 1
fi
if [ "$THREAD_POOL_SIZE" = "" ]; then
  echo Error, missing: THREAD_POOL_SIZE=6
  exit 1
fi

sudo DEBIAN_FRONTEND=noninteractive apt-get update -y


# Use $SSH_USER

if [ ! -d /home/$SSH_USER ]; then
  mkdir -p /home/$SSH_USER
fi
sudo chmod 755 /home/$SSH_USER



# Install Python libraries used by formulas



# Add all used Python modules here. # No need to install venv or pip, just proceed with python3-xyz
sudo DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=a apt-get install -y python3-dateutil




# Install Prometheus node exporter




sudo useradd --no-create-home --shell /bin/false node_exporter

if [ "$(uname -p)" == "x86_64" ]; then
  ARCH=amd64
else
  ARCH=arm64
fi

# Download node exporter
EXPORTER_DL_FILE=node_exporter-$PROMETHEUS_NODE_EXPORTER_VERSION.linux-$ARCH
cd /home/$SSH_USER
echo Downloading https://github.com/prometheus/node_exporter/releases/download/v$PROMETHEUS_NODE_EXPORTER_VERSION/$EXPORTER_DL_FILE.tar.gz ...
curl -LOs https://github.com/prometheus/node_exporter/releases/download/v$PROMETHEUS_NODE_EXPORTER_VERSION/$EXPORTER_DL_FILE.tar.gz
if [ "$?" -ne "0" ]; then
    echo Cannot download, exiting
    exit $?
fi
tar xvf $EXPORTER_DL_FILE.tar.gz


sudo cp $EXPORTER_DL_FILE/node_exporter /usr/local/bin
sudo chown node_exporter:node_exporter /usr/local/bin/node_exporter

rm -rf $EXPORTER_DL_FILE.tar.gz $EXPORTER_DL_FILE



# Configure Prometheus node exporter



# Make it as idempotent as possible, it can be called over and over

# Prometheus node exporter
# https://www.digitalocean.com/community/tutorials/how-to-install-prometheus-on-ubuntu-16-04

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




# Install capidaemon





CAPI_BINARY=capidaemon

if [ ! -d /home/$SSH_USER/bin ]; then
  mkdir -p /home/$SSH_USER/bin
fi
sudo chmod 755 /home/$SSH_USER/bin
cd /home/$SSH_USER/bin

curl -LOs $CAPILLARIES_RELEASE_URL/$OS_ARCH/$CAPI_BINARY.gz
if [ "$?" -ne "0" ]; then
    echo "Cannot download $CAPILLARIES_RELEASE_URL/$OS_ARCH/$CAPI_BINARY.gz to /home/$SSH_USER/bin"
    exit $?
fi
curl -LOs $CAPILLARIES_RELEASE_URL/$OS_ARCH/$CAPI_BINARY.json
if [ "$?" -ne "0" ]; then
    echo "Cannot download from $CAPILLARIES_RELEASE_URL/$OS_ARCH/$CAPI_BINARY.json to /home/$SSH_USER/bin"
    exit $?
fi
gzip -d -f $CAPI_BINARY.gz
chmod 744 $CAPI_BINARY

sudo mkdir /var/log/capillaries
sudo chown $SSH_USER /var/log/capillaries



# S3 log location



echo Checking access to $S3_LOG_URL...
aws s3 ls $S3_LOG_URL/

# Add hostname to the log file names and send them to S3 every 5 min
SEND_LOGS_FILE=/home/$SSH_USER/sendlogs.sh
sudo tee $SEND_LOGS_FILE <<EOF
#!/bin/bash
if [ -s /var/log/capillaries/capidaemon.log ]; then
  # Send SIGHUP to the running binary, it will rotate the log using Lumberjack
  ps axf | grep capidaemon | grep -v grep | awk '{print "kill -s 1 " \$1}' | sh
  for f in /var/log/capillaries/*.gz;do
    if [[ -e \$f && \$f=~^capi* ]]; then
      # Lumberjack produces: capidaemon-2025-05-03T21-37-01.283.log.gz
      # Add hostname to it: capidaemon-2025-05-03T21-37-01.283.ip-10-5-0-101.log.gz
      fname=\$(basename -- "\$f")
      fnamedatetime=\$(echo \$fname|cut -d'.' -f1)
      fnamemillis=\$(echo \$fname|cut -d'.' -f2)
      newfilepath=/var/log/capillaries/\$fnamedatetime.\$fnamemillis.\$HOSTNAME.log.gz
      mv \$f \$newfilepath
      aws s3 cp \$newfilepath $S3_LOG_URL/
      rm \$newfilepath
    fi
  done
fi
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

# Run daemon


# AWS region is required because S3 bucket pointer is a URI, not a URL
echo Running ...
echo 'kill -9 $(ps aux |grep capidaemon | grep bin | awk '"'"'{print $2}'"'"')'
echo GOMEMLIMIT="$DAEMON_GOMEMLIMIT_GB"GiB GOGC=$DAEMON_GOGC AWS_DEFAULT_REGION=$AWSREGION CAPI_PROMETHEUS_EXPORTER_PORT=9200 CAPI_CASSANDRA_HOSTS="$CASSANDRA_HOSTS" CAPI_CASSANDRA_PORT=$CASSANDRA_PORT CAPI_CASSANDRA_USERNAME=$CASSANDRA_USERNAME CAPI_CASSANDRA_PASSWORD=$CASSANDRA_PASSWORD CAPI_CASSANDRA_KEYSPACE_REPLICATION_CONFIG='"'"{'class':'NetworkTopologyStrategy','datacenter1':1}"'"' CAPI_CASSANDRA_CONSISTENCY=LOCAL_QUORUM CAPI_CASSANDRA_WRITER_WORKERS=$WRITER_WORKERS CAPI_THREAD_POOL_SIZE=$THREAD_POOL_SIZE CAPI_AMQP091_URL=$RABBITMQ_URL CAPI_CASSANDRA_TIMEOUT=15000 CAPI_PYCALC_INTERPRETER_PATH=python3 CAPI_LOG_LEVEL=info CAPI_DEAD_LETTER_TTL=5000 CAPI_LOG_FILE=/var/log/capillaries/capidaemon.log /home/$SSH_USER/bin/capidaemon "&>/dev/null &"
GOMEMLIMIT="$DAEMON_GOMEMLIMIT_GB"GiB GOGC=$DAEMON_GOGC AWS_DEFAULT_REGION=$AWSREGION CAPI_PROMETHEUS_EXPORTER_PORT=9200 CAPI_CASSANDRA_HOSTS="$CASSANDRA_HOSTS" CAPI_CASSANDRA_PORT=$CASSANDRA_PORT CAPI_CASSANDRA_USERNAME=$CASSANDRA_USERNAME CAPI_CASSANDRA_PASSWORD=$CASSANDRA_PASSWORD CAPI_CASSANDRA_KEYSPACE_REPLICATION_CONFIG="{'class':'NetworkTopologyStrategy','datacenter1':1}" CAPI_CASSANDRA_CONSISTENCY=LOCAL_QUORUM CAPI_CASSANDRA_WRITER_WORKERS=$WRITER_WORKERS CAPI_THREAD_POOL_SIZE=$THREAD_POOL_SIZE CAPI_AMQP091_URL=$RABBITMQ_URL CAPI_CASSANDRA_TIMEOUT=15000 CAPI_PYCALC_INTERPRETER_PATH=python3 CAPI_LOG_LEVEL=info CAPI_DEAD_LETTER_TTL=5000 CAPI_LOG_FILE="/var/log/capillaries/capidaemon.log" /home/$SSH_USER/bin/capidaemon &>/dev/null &
