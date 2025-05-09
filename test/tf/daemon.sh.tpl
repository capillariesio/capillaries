#!/bin/bash

echo Running daemon.sh.tpl in $(pwd)

sudo DEBIAN_FRONTEND=noninteractive apt-get update -y

# Add all used Python modules here. # No need to install venv or pip, just proceed with python3-xyz
sudo DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=a apt-get install -y python3-dateutil

# Use ${ssh_user}

if [ ! -d /home/${ssh_user} ]; then
  mkdir -p /home/${ssh_user}
fi
sudo chmod 755 /home/${ssh_user}


# Install capidaemon

CAPI_BINARY=capidaemon

if [ ! -d /home/${ssh_user}/bin ]; then
  mkdir -p /home/${ssh_user}/bin
fi
sudo chmod 755 /home/${ssh_user}/bin
cd /home/${ssh_user}/bin

curl -LOs ${capillaries_release_url}/${os_arch}/$CAPI_BINARY.gz
if [ "$?" -ne "0" ]; then
    echo "Cannot download ${capillaries_release_url}/${os_arch}/$CAPI_BINARY.gz to /home/${ssh_user}/bin"
    exit $?
fi
curl -LOs ${capillaries_release_url}/${os_arch}/$CAPI_BINARY.json
if [ "$?" -ne "0" ]; then
    echo "Cannot download from ${capillaries_release_url}/${os_arch}/$CAPI_BINARY.json to /home/${ssh_user}/bin"
    exit $?
fi
gzip -d -f $CAPI_BINARY.gz
chmod 744 $CAPI_BINARY

sudo mkdir /var/log/capillaries
sudo chown ${ssh_user} /var/log/capillaries


# AWS cert for Amazon Keyspaces

if [ ! -d /home/${ssh_user}/.ssh ]; then
  mkdir -p /home/${ssh_user}/.ssh
fi
sudo chmod 755 /home/${ssh_user}/.ssh
cd /home/${ssh_user}/.ssh

curl -LOs https://certs.secureserver.net/repository/sf-class2-root.crt
sudo chmod 644 ./sf-class2-root.crt


# CLI and S3 to send logs

sudo apt install unzip

if [ "${os_arch}" = "linux/arm64" ]; then
 curl "https://awscli.amazonaws.com/awscli-exe-linux-aarch64.zip" -o "awscliv2.zip"
fi
if [ "${os_arch}" = "linux/amd64" ]; then
 curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
fi

unzip awscliv2.zip
sudo ./aws/install
rm -fR aws
rm awscliv2.zip

# Test S3 log location
echo Checking access to ${s3_log_url}...
aws s3 ls ${s3_log_url}/

# Add hostname to the log file names and send them to S3 every 5 min
SEND_LOGS_FILE=/home/${ssh_user}/sendlogs.sh
sudo tee $SEND_LOGS_FILE <<EOF
#!/bin/bash
# Send SIGHUP to the running binary, it will rotate the log using Lumberjack
ps axf | grep capidaemon | grep -v grep | awk '{print "kill -s 1 " \$1}' | sh
for f in /var/log/capillaries/*.gz;do
  if [ -f \$f ]; then
    # Lumberjack produces: capidaemon-2025-05-03T21-37-01.283.log.gz
    # Add hostname to it: capidaemon-2025-05-03T21-37-01.283.ip-10-5-0-101.log.gz
    fname=\$(basename -- "\$f")
    fnamedatetime=\$(echo \$fname|cut -d'.' -f1)
    fnamemillis=\$(echo \$fname|cut -d'.' -f2)
    newfilepath=/var/log/capillaries/\$fnamedatetime.\$fnamemillis.\$HOSTNAME.log.gz
    mv \$f \$newfilepath
    aws s3 cp \$newfilepath ${s3_log_url}/
    rm \$newfilepath
  fi
done
EOF
sudo chmod 744 $SEND_LOGS_FILE
sudo su ${ssh_user} -c "echo \"*/5 * * * * $SEND_LOGS_FILE\" | crontab -"


# Everything in ~ should belong to ssh user
sudo chown -R ${ssh_user} /home/${ssh_user}


# Run daemon

# AWS region is required because S3 bucket pointer is a URI, not a URL
sudo su ${ssh_user} -c 'AWS_DEFAULT_REGION=${awsregion} CAPI_CASSANDRA_HOSTS="${cassandra_hosts}" CAPI_CASSANDRA_PORT=${cassandra_port} CAPI_CASSANDRA_USERNAME="${cassandra_username}" CAPI_CASSANDRA_PASSWORD="${cassandra_password}" CAPI_CASSANDRA_CA_PATH="/home/${ssh_user}/.ssh/sf-class2-root.crt" CAPI_CASSANDRA_ENABLE_HOST_VERIFICATION=false CAPI_CASSANDRA_KEYSPACE_REPLICATION_CONFIG="'"{ 'class' : 'SingleRegionStrategy'}"'" CAPI_CASSANDRA_CONSISTENCY=LOCAL_QUORUM CAPI_CASSANDRA_WRITER_WORKERS=20 CAPI_CASSANDRA_MIN_INSERTER_RATE=5 CAPI_THREAD_POOL_SIZE=8 CAPI_AMQP091_URL="${rabbitmq_url}" CAPI_CASSANDRA_TIMEOUT=15000 CAPI_PYCALC_INTERPRETER_PATH=python3 CAPI_LOG_LEVEL=info CAPI_LOG_FILE="/var/log/capillaries/capidaemon.log" /home/${ssh_user}/bin/capidaemon &>/dev/null &'
