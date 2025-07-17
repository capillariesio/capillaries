#!/bin/bash

echo Running bastion.sh in $(pwd) ...

if [ "$WEBAPI_GOMEMLIMIT_GB" = "" ]; then
  echo Error, missing: WEBAPI_GOMEMLIMIT_GB=2
  exit 1
fi
if [ "$WEBAPI_GOGC" = "" ]; then
  echo Error, missing: WEBAPI_GOGC=100
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
if [ "$BASTION_EXTERNAL_IP_ADDRESS" = "" ]; then
  echo Error, missing: BASTION_EXTERNAL_IP_ADDRESS=...
  exit 1
fi
if [ "$EXTERNAL_WEBAPI_PORT" = "" ]; then
  echo Error, missing: EXTERNAL_WEBAPI_PORT=6544
  exit 1
fi
if [ "$INTERNAL_WEBAPI_PORT" = "" ]; then
  echo Error, missing: INTERNAL_WEBAPI_PORT=6543
  exit 1
fi
if [ "$OS_ARCH" = "" ]; then
  echo Error, missing: OS_ARCH=linux/arm64
  exit 1
fi
if [ "$EXTERNAL_RABBITMQ_CONSOLE_PORT" = "" ]; then
  echo Error, missing: EXTERNAL_RABBITMQ_CONSOLE_PORT=15673
  exit 1
fi
if [ "$EXTERNAL_PROMETHEUS_CONSOLE_PORT" = "" ]; then
  echo Error, missing: EXTERNAL_PROMETHEUS_CONSOLE_PORT=9091
  exit 1
fi
if [ "$BASTION_ALLOWED_IPS" = "" ]; then
  echo Error, missing: BASTION_ALLOWED_IPS=1.2.3.4,1.2.0.0/16
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
if [ "$RABBITMQ_USER_NAME" = "" ]; then
  echo Error, missing: RABBITMQ_USER_NAME=capiuser
  exit 1
fi
if [ "$RABBITMQ_USER_PASS" = "" ]; then
  echo Error, missing: RABBITMQ_USER_PASS=capipass
  exit 1
fi
if [ "$RABBITMQ_ADMIN_NAME" = "" ]; then
  echo Error, missing: RABBITMQ_ADMIN_NAME=radmin
  exit 1
fi
if [ "$RABBITMQ_ADMIN_PASS" = "" ]; then
  echo Error, missing: RABBITMQ_ADMIN_PASS=rpass
  exit 1
fi
if [ "$RABBITMQ_ERLANG_VERSION_AMD64" = "" ]; then
  echo Error, missing: RABBITMQ_ERLANG_VERSION_AMD64=1:27.2-1
  exit 1
fi
if [ "$RABBITMQ_SERVER_VERSION_AMD64" = "" ]; then
  echo Error, missing: RABBITMQ_SERVER_VERSION_AMD64=4.0.5-1
  exit 1
fi
if [ "$RABBITMQ_ERLANG_VERSION_ARM64" = "" ]; then
  echo Error, missing: RABBITMQ_ERLANG_VERSION_ARM64=1:25.3.2.8+dfsg-1ubuntu4
  exit 1
fi
if [ "$RABBITMQ_SERVER_VERSION_ARM64" = "" ]; then
  echo Error, missing: RABBITMQ_SERVER_VERSION_ARM64=3.12.1-1ubuntu1
  exit 1
fi
if [ "$PROMETHEUS_NODE_EXPORTER_VERSION" = "" ]; then
  echo Error, missing: PROMETHEUS_NODE_EXPORTER_VERSION=1.2.3
  exit 1
fi
if [ "$PROMETHEUS_SERVER_VERSION" = "" ]; then
  echo Error, missing: PROMETHEUS_SERVER_VERSION=1.2.3
  exit 1
fi
if [ "$PROMETHEUS_NODE_TARGETS" = "" ]; then
  echo Error, missing: PROMETHEUS_NODE_TARGETS="'localhost:9100','10.5.1.10:9100'"
  exit 1
fi
if [ "$PROMETHEUS_JMX_TARGETS" = "" ]; then
  echo Error, missing: PROMETHEUS_JMX_TARGETS="'10.5.1.11:7070','10.5.1.12:7070'"
  exit 1
fi
if [ "$PROMETHEUS_GO_TARGETS" = "" ]; then
  echo Error, missing: PROMETHEUS_GO_TARGETS="'10.5.1.101:9200','10.5.1.102:9200'"
  exit 1
fi

sudo DEBIAN_FRONTEND=noninteractive apt-get update -y


# Use $SSH_USER

if [ ! -d /home/$SSH_USER ]; then
  mkdir -p /home/$SSH_USER
fi
sudo chmod 755 /home/$SSH_USER



# Install nginx



sudo DEBIAN_FRONTEND=noninteractive apt-get install -y nginx
if [ "$?" -ne "0" ]; then
    echo nginx install error, exiting
    exit $?
fi

# Remove nginx stub site
sudo rm -f /etc/nginx/sites-enabled/default

sudo nginx -t
if [ "$?" -ne "0" ]; then
    echo nginx config error, exiting
    exit $?
fi

sudo systemctl restart nginx



# Config IP address whitelist (make sure it's before reverse proxies that use it)



if [ ! -d "/etc/nginx/includes" ]; then
  sudo mkdir /etc/nginx/includes
fi

WHITELIST_CONFIG_FILE=/etc/nginx/includes/allowed_ips.conf

if [ -f "$WHITELIST_CONFIG_FILE" ]; then
  sudo rm $WHITELIST_CONFIG_FILE
fi
sudo touch $WHITELIST_CONFIG_FILE

IFS=',' read -ra CIDR <<< "$BASTION_ALLOWED_IPS"
for i in "${CIDR[@]}"; do
  echo "allow $i;" | sudo tee -a $WHITELIST_CONFIG_FILE
done
echo "deny all;" | sudo tee -a $WHITELIST_CONFIG_FILE




# Install UI




if [ ! -d /home/$SSH_USER/ui ]; then
  mkdir -p /home/$SSH_USER/ui
fi
sudo chmod 755 /home/$SSH_USER/ui
cd /home/$SSH_USER/ui

curl -LOs $CAPILLARIES_RELEASE_URL/webui/webui.tgz
if [ "$?" -ne "0" ]; then
    echo "Cannot download webui from $CAPILLARIES_RELEASE_URL/webui/webui.tgz to /home/$SSH_USER/ui"
    exit $?
fi

tar xvzf webui.tgz
rm webui.tgz

# Tweak UI so it calls the proper capiwebapi URL
# This is not idempotent. It's actually pretty hacky.
echo Patching WebUI to use external Webapi ip:port $BASTION_EXTERNAL_IP_ADDRESS:$EXTERNAL_WEBAPI_PORT
sed -i -e 's~localhost:6543~'$BASTION_EXTERNAL_IP_ADDRESS':'$EXTERNAL_WEBAPI_PORT'~g' /home/$SSH_USER/ui/_app/immutable/chunks/*.js



# Install Webapi



CAPI_BINARY=capiwebapi

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
sudo chmod 744 $CAPI_BINARY

sudo mkdir /var/log/capillaries
sudo chown $SSH_USER /var/log/capillaries



# Reverse proxies and servers


# UI server
UI_CONFIG_FILE=/etc/nginx/sites-available/ui
if [ -f "$UI_CONFIG_FILE" ]; then
  sudo rm -f $UI_CONFIG_FILE
fi

sudo tee $UI_CONFIG_FILE <<EOF
server {
  listen 80;
  listen [::]:80;
  root /home/$SSH_USER/ui;
  index index.html;
  location / {
    include includes/allowed_ips.conf;
  }
}
EOF

if [ ! -L "/etc/nginx/sites-enabled/ui" ]; then
  sudo ln -s $UI_CONFIG_FILE /etc/nginx/sites-enabled/
fi

sudo nginx -t
if [ "$?" -ne "0" ]; then
    echo nginx ui config error, exiting
    exit $?
fi


# Webapi reverse proxy
WEBAPI_CONFIG_FILE=/etc/nginx/sites-available/webapi
if [ -f "$WEBAPI_CONFIG_FILE" ]; then
  sudo rm -f $WEBAPI_CONFIG_FILE
fi

sudo tee $WEBAPI_CONFIG_FILE <<EOF
server {
    listen $EXTERNAL_WEBAPI_PORT;
    location / {
        proxy_pass http://localhost:$INTERNAL_WEBAPI_PORT;
        include proxy_params;
        include includes/allowed_ips.conf;
    }
}
EOF

if [ ! -L "/etc/nginx/sites-enabled/webapi" ]; then
  sudo ln -s $WEBAPI_CONFIG_FILE /etc/nginx/sites-enabled/
fi

sudo nginx -t
if [ "$?" -ne "0" ]; then
    echo nginx webapi reverse proxy config error, exiting
    exit $?
fi


# RabbitMQ reverse proxy
RABBITMQ_CONFIG_FILE=/etc/nginx/sites-available/rabbitmq
if [ -f "$RABBITMQ_CONFIG_FILE" ]; then
  sudo rm -f $RABBITMQ_CONFIG_FILE
fi

sudo tee $RABBITMQ_CONFIG_FILE <<EOF
server {
    listen $EXTERNAL_RABBITMQ_CONSOLE_PORT;
    location / {
        proxy_pass http://localhost:15672;
        include proxy_params;
        include includes/allowed_ips.conf;
    }
}
EOF

if [ ! -L "/etc/nginx/sites-enabled/rabbitmq" ]; then
  sudo ln -s $RABBITMQ_CONFIG_FILE /etc/nginx/sites-enabled/
fi

sudo nginx -t
if [ "$?" -ne "0" ]; then
    echo nginx rabbitmq reverse proxy config error, exiting
    exit $?
fi

# Prometheus reverse proxy
PROMETHEUS_CONFIG_FILE=/etc/nginx/sites-available/prometheus
if [ -f "$PROMETHEUS_CONFIG_FILE" ]; then
  sudo rm -f $PROMETHEUS_CONFIG_FILE
fi

sudo tee $PROMETHEUS_CONFIG_FILE <<EOF
server {
    listen $EXTERNAL_PROMETHEUS_CONSOLE_PORT;
    location / {
        proxy_pass http://localhost:9090;
        include proxy_params;
        include includes/allowed_ips.conf;
    }
}
EOF

if [ ! -L "/etc/nginx/sites-enabled/prometheus" ]; then
  sudo ln -s $PROMETHEUS_CONFIG_FILE /etc/nginx/sites-enabled/
fi

sudo nginx -t
if [ "$?" -ne "0" ]; then
    echo nginx prometheus reverse proxy config error, exiting
    exit $?
fi



# Restart nginx to pick up changes

sudo nginx -t
if [ "$?" -ne "0" ]; then
    echo nginx config error, exiting
    exit $?
fi

sudo systemctl restart nginx




# S3 log location




echo Checking access to $S3_LOG_URL...
aws s3 ls $S3_LOG_URL/

# Add hostname to the log file names and send them to S3 every 5 min
SEND_LOGS_FILE=/home/$SSH_USER/sendlogs.sh
sudo tee $SEND_LOGS_FILE <<EOF
#!/bin/bash
if [ -s /var/log/capillaries/capiwebapi.log ]; then
  # Send SIGHUP to the running binary, it will rotate the log using Lumberjack
  ps axf | grep capiwebapi | grep -v grep | awk '{print "kill -s 1 " \$1}' | sh
  for f in /var/log/capillaries/*.gz;do
    if [[ -e \$f && \$f=~^capi* ]]; then
      # Lumberjack produces: capiwebapi-2025-05-03T21-37-01.283.log.gz
      # Add hostname to it: capiwebapi-2025-05-03T21-37-01.283.ip-10-5-1-10.log.gz
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


# Run Webapi

# If we ever use https and/or domain names, or use other port than 80, revisit this piece.
# AWS region is required because S3 bucket pointer is a URI, not a URL
echo Running webapi with GOMEMLIMIT="$WEBAPI_GOMEMLIMIT_GB"GiB GOGC=$WEBAPI_GOGC AWS_DEFAULT_REGION=$AWSREGION CAPI_PROMETHEUS_EXPORTER_PORT=9200 CAPI_WEBAPI_ACCESS_CONTROL_ALLOW_ORIGIN="http://$BASTION_EXTERNAL_IP_ADDRESS" CAPI_WEBAPI_PORT=$INTERNAL_WEBAPI_PORT CAPI_CASSANDRA_HOSTS="$CASSANDRA_HOSTS" CAPI_CASSANDRA_PORT=$CASSANDRA_PORT CAPI_CASSANDRA_USERNAME="$CASSANDRA_USERNAME" CAPI_CASSANDRA_PASSWORD="$CASSANDRA_PASSWORD" CAPI_CASSANDRA_ENABLE_HOST_VERIFICATION=false CAPI_CASSANDRA_KEYSPACE_REPLICATION_CONFIG="{ 'class' : 'NetworkTopologyStrategy', 'datacenter1' : 1 }" CAPI_CASSANDRA_CONSISTENCY=LOCAL_QUORUM CAPI_AMQP091_URL="$RABBITMQ_URL" CAPI_CASSANDRA_TIMEOUT=15000 CAPI_LOG_LEVEL=info CAPI_LOG_FILE="/var/log/capillaries/capiwebapi.log"
echo To stop it: 'kill -9 $(ps aux |grep capiwebapi | grep bin | awk '"'"'{print $2}'"'"')'
GOMEMLIMIT="$WEBAPI_GOMEMLIMIT_GB"GiB GOGC=$WEBAPI_GOGC AWS_DEFAULT_REGION=$AWSREGION CAPI_PROMETHEUS_EXPORTER_PORT=9200 CAPI_WEBAPI_ACCESS_CONTROL_ALLOW_ORIGIN="http://$BASTION_EXTERNAL_IP_ADDRESS" CAPI_WEBAPI_PORT=$INTERNAL_WEBAPI_PORT CAPI_CASSANDRA_HOSTS="$CASSANDRA_HOSTS" CAPI_CASSANDRA_PORT=$CASSANDRA_PORT CAPI_CASSANDRA_USERNAME="$CASSANDRA_USERNAME" CAPI_CASSANDRA_PASSWORD="$CASSANDRA_PASSWORD" CAPI_CASSANDRA_ENABLE_HOST_VERIFICATION=false CAPI_CASSANDRA_KEYSPACE_REPLICATION_CONFIG="{ 'class' : 'NetworkTopologyStrategy', 'datacenter1' : 1 }" CAPI_CASSANDRA_CONSISTENCY=LOCAL_QUORUM CAPI_AMQP091_URL="$RABBITMQ_URL" CAPI_CASSANDRA_TIMEOUT=15000 CAPI_LOG_LEVEL=info CAPI_LOG_FILE="/var/log/capillaries/capiwebapi.log" /home/$SSH_USER/bin/capiwebapi &>/dev/null &





# Install RabbitMQ






sudo DEBIAN_FRONTEND=noninteractive apt-get install -y curl gnupg
if [ "$?" -ne "0" ]; then
    echo gnugpg install error, exiting
    exit $?
fi

sudo DEBIAN_FRONTEND=noninteractive apt-get install -y apt-transport-https
if [ "$?" -ne "0" ]; then
    echo apt-transport-https install error, exiting
    exit $?
fi

## Team RabbitMQ's main signing key
curl -1sLf "https://keys.openpgp.org/vks/v1/by-fingerprint/0A9AF2115F4687BD29803A206B73A36E6026DFCA" | sudo gpg --dearmor | sudo tee /usr/share/keyrings/com.rabbitmq.team.gpg > /dev/null
## Community mirror of Cloudsmith: modern Erlang repository
curl -1sLf https://github.com/rabbitmq/signing-keys/releases/download/3.0/cloudsmith.rabbitmq-erlang.E495BB49CC4BBE5B.key | sudo gpg --dearmor | sudo tee /usr/share/keyrings/rabbitmq.E495BB49CC4BBE5B.gpg > /dev/null
## Community mirror of Cloudsmith: RabbitMQ repository
curl -1sLf https://github.com/rabbitmq/signing-keys/releases/download/3.0/cloudsmith.rabbitmq-server.9F4587F226208342.key | sudo gpg --dearmor | sudo tee /usr/share/keyrings/rabbitmq.9F4587F226208342.gpg > /dev/null


sudo tee /etc/apt/sources.list.d/rabbitmq.list <<EOF
## Provides modern Erlang/OTP releases from a Cloudsmith mirror
##
deb [arch=amd64 signed-by=/usr/share/keyrings/rabbitmq.E495BB49CC4BBE5B.gpg] https://ppa1.rabbitmq.com/rabbitmq/rabbitmq-erlang/deb/ubuntu noble main
deb-src [signed-by=/usr/share/keyrings/rabbitmq.E495BB49CC4BBE5B.gpg] https://ppa1.rabbitmq.com/rabbitmq/rabbitmq-erlang/deb/ubuntu noble main

# another mirror for redundancy
deb [arch=amd64 signed-by=/usr/share/keyrings/rabbitmq.E495BB49CC4BBE5B.gpg] https://ppa2.rabbitmq.com/rabbitmq/rabbitmq-erlang/deb/ubuntu noble main
deb-src [signed-by=/usr/share/keyrings/rabbitmq.E495BB49CC4BBE5B.gpg] https://ppa2.rabbitmq.com/rabbitmq/rabbitmq-erlang/deb/ubuntu noble main

## Provides RabbitMQ from a Cloudsmith mirror
##
deb [arch=amd64 signed-by=/usr/share/keyrings/rabbitmq.9F4587F226208342.gpg] https://ppa1.rabbitmq.com/rabbitmq/rabbitmq-server/deb/ubuntu noble main
deb-src [signed-by=/usr/share/keyrings/rabbitmq.9F4587F226208342.gpg] https://ppa1.rabbitmq.com/rabbitmq/rabbitmq-server/deb/ubuntu noble main

# another mirror for redundancy
deb [arch=amd64 signed-by=/usr/share/keyrings/rabbitmq.9F4587F226208342.gpg] https://ppa2.rabbitmq.com/rabbitmq/rabbitmq-server/deb/ubuntu noble main
deb-src [signed-by=/usr/share/keyrings/rabbitmq.9F4587F226208342.gpg] https://ppa2.rabbitmq.com/rabbitmq/rabbitmq-server/deb/ubuntu noble main
EOF

sudo DEBIAN_FRONTEND=noninteractive apt-get update -y

# See available packages:

# apt list -a erlang-base

# As of Dec 2024:
# erlang-base/noble,noble 1:27.2-1 amd64
# erlang-base/noble,noble 1:27.1.3-1 amd64
# erlang-base/noble,noble 1:27.1.2-1 amd64
# erlang-base/noble,noble 1:26.2.5.6-1 amd64
# erlang-base/noble,noble 1:26.2.5.5-1 amd64
# erlang-base/noble,noble 1:26.2.5.4-1 amd64
# erlang-base/noble,now 1:25.3.2.8+dfsg-1ubuntu4 arm64 [installed]

# As of April 2025:
# erlang-base/noble,noble 1:27.3.1-1 amd64
# erlang-base/noble,noble 1:27.3-1 amd64
# erlang-base/noble,noble 1:27.2.4-1 amd64
# erlang-base/noble,noble 1:27.2.3-1 amd64
# erlang-base/noble,noble 1:27.2.2-1 amd64
# erlang-base/noble,noble 1:27.2.1-1 amd64
# erlang-base/noble,noble 1:27.2-1 amd64
# erlang-base/noble,noble 1:27.1.3-1 amd64
# erlang-base/noble,noble 1:27.1.2-1 amd64
# erlang-base/noble,noble 1:26.2.5.10-1 amd64
# erlang-base/noble,noble 1:26.2.5.9-1 amd64
# erlang-base/noble,noble 1:26.2.5.8-1 amd64
# erlang-base/noble,noble 1:26.2.5.7-1 amd64
# erlang-base/noble,noble 1:26.2.5.6-1 amd64
# erlang-base/noble,noble 1:26.2.5.5-1 amd64
# erlang-base/noble,noble 1:26.2.5.4-1 amd64
# erlang-base/noble-updates,noble-security,now 1:25.3.2.8+dfsg-1ubuntu4.1 arm64 [installed]
# erlang-base/noble 1:25.3.2.8+dfsg-1ubuntu4 arm64

# As of April 2025
# erlang-base/noble,noble 1:27.3.2-1 amd64
# erlang-base/noble,noble 1:27.3.1-1 amd64
# erlang-base/noble,noble 1:27.3-1 amd64
# erlang-base/noble,noble 1:27.2.4-1 amd64
# erlang-base/noble,noble 1:27.2.3-1 amd64
# erlang-base/noble,noble 1:27.2.2-1 amd64
# erlang-base/noble,noble 1:27.2.1-1 amd64
# erlang-base/noble,noble 1:27.2-1 amd64
# erlang-base/noble,noble 1:27.1.3-1 amd64
# erlang-base/noble,noble 1:27.1.2-1 amd64
# erlang-base/noble,noble 1:26.2.5.10-1 amd64
# erlang-base/noble,noble 1:26.2.5.9-1 amd64
# erlang-base/noble,noble 1:26.2.5.8-1 amd64
# erlang-base/noble,noble 1:26.2.5.7-1 amd64
# erlang-base/noble,noble 1:26.2.5.6-1 amd64
# erlang-base/noble,noble 1:26.2.5.5-1 amd64
# erlang-base/noble,noble 1:26.2.5.4-1 amd64
# erlang-base/noble-updates,noble-security 1:25.3.2.8+dfsg-1ubuntu4.2 arm64 [upgradable from: 1:25.3.2.8+dfsg-1ubuntu4]
# erlang-base/noble,now 1:25.3.2.8+dfsg-1ubuntu4 arm64 [installed,upgradable to: 1:25.3.2.8+dfsg-1ubuntu4.2]

# apt list -a rabbitmq-server

# As of Dec 2024:
# rabbitmq-server/noble,noble 4.0.5-1 all [upgradable from: 3.12.1-1ubuntu1]
# rabbitmq-server/noble,noble 4.0.4-1 all
# rabbitmq-server/noble,noble 4.0.3-1 all
# rabbitmq-server/noble,noble 4.0.2-1 all
# rabbitmq-server/noble,noble 4.0.1-1 all
# rabbitmq-server/noble,noble 4.0.0-1 all
# rabbitmq-server/noble,noble 3.13.7-1 all
# rabbitmq-server/noble,noble 3.13.6-1 all
# rabbitmq-server/noble,noble 3.13.5-1 all
# rabbitmq-server/noble,noble 3.13.4-1 all
# rabbitmq-server/noble,noble 3.12.14-1 all
# rabbitmq-server/noble,now 3.12.1-1ubuntu1 all [installed,upgradable to: 4.0.5-1]

# As of April 2025:
# rabbitmq-server/noble,noble 4.0.7-1 all [upgradable from: 3.12.1-1ubuntu1]
# rabbitmq-server/noble,noble 4.0.6-1 all
# rabbitmq-server/noble,noble 4.0.5-1 all
# rabbitmq-server/noble,noble 4.0.4-1 all
# rabbitmq-server/noble,noble 4.0.3-1 all
# rabbitmq-server/noble,noble 4.0.2-1 all
# rabbitmq-server/noble,noble 4.0.1-1 all
# rabbitmq-server/noble,noble 4.0.0-1 all
# rabbitmq-server/noble,noble 3.13.7-1 all
# rabbitmq-server/noble,noble 3.13.6-1 all
# rabbitmq-server/noble,noble 3.13.5-1 all
# rabbitmq-server/noble,noble 3.13.4-1 all
# rabbitmq-server/noble,noble 3.12.14-1 all
# rabbitmq-server/noble-updates,noble-security 3.12.1-1ubuntu1.2 all
# rabbitmq-server/noble,now 3.12.1-1ubuntu1 all [installed,upgradable to: 4.0.7-1]

# Compatibility chart: https://www.rabbitmq.com/docs/which-erlang and https://www.rabbitmq.com/docs/3.13/which-erlang

if [ "$(uname -p)" == "x86_64" ]; then
  export ERLANG_VER=$RABBITMQ_ERLANG_VERSION_AMD64
  export RABBITMQ_VER=$RABBITMQ_SERVER_VERSION_AMD64
else
  export ERLANG_VER=$RABBITMQ_ERLANG_VERSION_ARM64
  export RABBITMQ_VER=$RABBITMQ_SERVER_VERSION_ARM64
fi

sudo DEBIAN_FRONTEND=noninteractive apt-get install -y erlang-base=$ERLANG_VER \
                        erlang-asn1=$ERLANG_VER erlang-crypto=$ERLANG_VER erlang-eldap=$ERLANG_VER erlang-ftp=$ERLANG_VER erlang-inets=$ERLANG_VER \
                        erlang-mnesia=$ERLANG_VER erlang-os-mon=$ERLANG_VER erlang-parsetools=$ERLANG_VER erlang-public-key=$ERLANG_VER \
                        erlang-runtime-tools=$ERLANG_VER erlang-snmp=$ERLANG_VER erlang-ssl=$ERLANG_VER \
                        erlang-syntax-tools=$ERLANG_VER erlang-tftp=$ERLANG_VER erlang-tools=$ERLANG_VER erlang-xmerl=$ERLANG_VER
if [ "$?" -ne "0" ]; then
    echo erlang install error, exiting
    exit $?
fi

sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --fix-missing rabbitmq-server=$RABBITMQ_VER
if [ "$?" -ne "0" ]; then
    echo rabbitmq install error, exiting
    exit $?
fi




# Configure RabbitMQ



# Make sure it's stopped
sudo systemctl stop rabbitmq-server

sudo tee /etc/rabbitmq/rabbitmq.conf <<EOF
# log.file=/var/log/rabbitmq/rabbit.log
# log.file.level=info
# log.file.formatter=json
log.file.rotation.date = \$D0
log.file.rotation.count = 5
log.file.rotation.compress = true
EOF

sudo chown rabbitmq /etc/rabbitmq/rabbitmq.conf
sudo chmod 644 /etc/rabbitmq/rabbitmq.conf

# Make sure it's started
sudo systemctl start rabbitmq-server

# Enable mgmt console
sudo rabbitmq-plugins list
sudo rabbitmq-plugins enable rabbitmq_management

# Console user mgmt
sudo rabbitmqctl add_user $RABBITMQ_ADMIN_NAME $RABBITMQ_ADMIN_PASS
sudo rabbitmqctl set_user_tags $RABBITMQ_ADMIN_NAME administrator
sudo rabbitmqctl set_permissions -p / $RABBITMQ_ADMIN_NAME ".*" ".*" ".*"

# Delete default guest user
sudo rabbitmqctl list_users
sudo rabbitmqctl delete_user guest

# Capillaries daemon and webapi use this account
sudo rabbitmqctl add_user $RABBITMQ_USER_NAME $RABBITMQ_USER_PASS
sudo rabbitmqctl set_permissions -p / $RABBITMQ_USER_NAME ".*" ".*" ".*"

curl -s http://localhost:15672
if [ "$?" -ne "0" ]; then
    echo Cannot check localhost:15672
    exit $?
fi



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





# Install Prometheus server





# Create users
sudo useradd --no-create-home --shell /bin/false prometheus

# Before we download the Prometheus binaries, create the necessary directories for storing Prometheus’ files and data. Following standard Linux conventions, we’ll create a directory in /etc for Prometheus’ configuration files and a directory in /var/lib for its data.
sudo mkdir /etc/prometheus
sudo mkdir /var/lib/prometheus

# Now, set the user and group ownership on the new directories to the prometheus user.
sudo chown prometheus:prometheus /etc/prometheus
sudo chown prometheus:prometheus /var/lib/prometheus

if [ "$(uname -p)" == "x86_64" ]; then
  ARCH=amd64
else
  ARCH=arm64
fi

# Downloading Prometheus
PROMETHEUS_DL_FILE=prometheus-$PROMETHEUS_SERVER_VERSION.linux-$ARCH
cd /home/$SSH_USER
echo Downloading https://github.com/prometheus/prometheus/releases/download/v$PROMETHEUS_SERVER_VERSION/$PROMETHEUS_DL_FILE.tar.gz
curl -LOs https://github.com/prometheus/prometheus/releases/download/v$PROMETHEUS_SERVER_VERSION/$PROMETHEUS_DL_FILE.tar.gz
if [ "$?" -ne "0" ]; then
    echo Cannot download, exiting
    exit $?
fi
tar xvf $PROMETHEUS_DL_FILE.tar.gz

# Copy the two binaries to the /usr/local/bin directory.

sudo cp $PROMETHEUS_DL_FILE/prometheus /usr/local/bin/
sudo cp $PROMETHEUS_DL_FILE/promtool /usr/local/bin/

# Set the user and group ownership on the binaries to the prometheus user created in Step 1.
sudo chown prometheus:prometheus /usr/local/bin/prometheus
sudo chown prometheus:prometheus /usr/local/bin/promtool

# Copy the consoles and console_libraries directories to /etc/prometheus.
# Not in 3.2.1
#sudo cp -r $PROMETHEUS_DL_FILE/consoles /etc/prometheus
#sudo cp -r $PROMETHEUS_DL_FILE/console_libraries /etc/prometheus

# Set the user and group ownership on the directories to the prometheus user. Using the -R flag will ensure that ownership is set on the files inside the directory as well.
# Not in 3.2.1
#sudo chown -R prometheus:prometheus /etc/prometheus/consoles
#sudo chown -R prometheus:prometheus /etc/prometheus/console_libraries

# Lastly, remove the leftover files from your home directory as they are no longer needed.
rm -rf $PROMETHEUS_DL_FILE.tar.gz $PROMETHEUS_DL_FILE




# Configure Prometheus server



# Prometheus server (assuming node exporter also running on it)
# https://www.digitalocean.com/community/tutorials/how-to-install-prometheus-on-ubuntu-16-04

sudo systemctl stop prometheus

PROMETHEUS_YAML_FILE=/etc/prometheus/prometheus.yml

sudo rm -f $PROMETHEUS_YAML_FILE

sudo tee $PROMETHEUS_YAML_FILE <<EOF
global:
  scrape_interval: 15s
scrape_configs:
  - job_name: 'node_exporter'
    scrape_interval: 5s
    static_configs:
      - targets: [$PROMETHEUS_NODE_TARGETS]
  - job_name: 'jmx_exporter'
    scrape_interval: 5s
    static_configs:
      - targets: [$PROMETHEUS_JMX_TARGETS]
  - job_name: 'go_exporter'
    scrape_interval: 5s
    static_configs:
      - targets: [$PROMETHEUS_GO_TARGETS]
EOF
sudo chown -R prometheus:prometheus $PROMETHEUS_YAML_FILE

PROMETHEUS_SERVICE_FILE=/etc/systemd/system/prometheus.service

sudo rm -f $PROMETHEUS_SERVICE_FILE

sudo tee $PROMETHEUS_SERVICE_FILE <<EOF
[Unit] 
Description=Prometheus server
Wants=network-online.target
After=network-online.target
[Service]
User=prometheus
Group=prometheus
Type=simple
ExecStart=/usr/local/bin/prometheus --config.file /etc/prometheus/prometheus.yml --storage.tsdb.path /var/lib/prometheus/ --storage.tsdb.retention.time=1d
[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload

sudo systemctl start prometheus
sudo systemctl status prometheus

# Check node exporter, by now it should be up
curl -s http://localhost:9100/metrics >/dev/null
if [ "$?" -ne "0" ]; then
    echo localhost:9100/metrics failed with $?
fi

# Check Prometheus UI
curl -s http://localhost:9090
if [ "$?" -ne "0" ]; then
    echo Cannot check localhost:9090
    exit $?
fi
