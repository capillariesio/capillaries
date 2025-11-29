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
if [ "$AWSREGION" = "" ]; then
  echo Error, missing: AWSREGION=us-east-1
  exit 1
fi
if [ "$SSH_USER" = "" ]; then
  echo Error, missing: SSH_USER=ubuntu
  exit 1
fi
if [ "$OS_ARCH" = "" ]; then
  echo Error, missing: OS_ARCH=linux/arm64
  exit 1
fi
if [ "$CAPILLARIES_RELEASE_URL" = "" ]; then
  echo Error, missing: CAPILLARIES_RELEASE_URL=https://capillaries-release.s3.us-east-1.amazonaws.com/latest
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
if [ "$EXTERNAL_RABBITMQ_CONSOLE_PORT" = "" ]; then
  echo Error, missing: EXTERNAL_RABBITMQ_CONSOLE_PORT=15673
  exit 1
fi
if [ "$EXTERNAL_ACTIVEMQ_CONSOLE_PORT" = "" ]; then
  echo Error, missing: EXTERNAL_ACTIVEMQ_CONSOLE_PORT=8162
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
if [ "$MQ_TYPE" = "" ]; then
  echo Error, missing: MQ_TYPE=amqp10
  exit 1
fi
if [ "$CAPIMQ_CLIENT_URL" = "" ]; then
  echo Error, missing: CAPIMQ_CLIENT_URL=http://10.5.1.10:7654
  exit 1
fi
if [ "$AMQP10_URL" = "" ]; then
  echo Error, missing: AMQP10_URL=amqps://capiuser:capiuserpass@10.5.1.10/
  exit 1
fi
if [ "$AMQP10_ADDRESS" = "" ]; then
  echo Error, missing: AMQP10_ADDRESS=/queue/capidaemon
  exit 1
fi
if [ "$AMQP10_USER_NAME" = "" ]; then
  echo Error, missing: AMQP10_USER_NAME=capiuser
  exit 1
fi
if [ "$AMQP10_USER_PASS" = "" ]; then
  echo Error, missing: AMQP10_USER_PASS=capuseripass
  exit 1
fi
if [ "$AMQP10_ADMIN_NAME" = "" ]; then
  echo Error, missing: AMQP10_ADMIN_NAME=capiadmin
  exit 1
fi
if [ "$AMQP10_ADMIN_PASS" = "" ]; then
  echo Error, missing: AMQP10_ADMIN_PASS=capiadminpass
  exit 1
fi
if [ "$AMQP10_SERVER_FLAVOR" = "" ]; then
  echo Error, missing: AMQP10_SERVER_FLAVOR=rabbitmq
  exit 1
fi
if [ "$PROMETHEUS_NODE_EXPORTER_FILENAME" = "" ]; then
  echo Error, missing: PROMETHEUS_NODE_EXPORTER_FILENAME=node_exporter-1.9.1.linux-amd64.tar.gz
  exit 1
fi
if [ "$PROMETHEUS_SERVER_FILENAME" = "" ]; then
  echo Error, missing: PROMETHEUS_SERVER_FILENAME=prometheus-3.7.0.linux-arm64.tar.gz
  exit 1
fi
if [ "$PROMETHEUS_NODE_TARGETS" = "" ]; then
  echo Error, missing: PROMETHEUS_NODE_TARGETS="'localhost:9100','10.5.1.10:9100'"
  exit 1
fi
if [ "$PROMETHEUS_JMX_TARGETS" = "" ]; then
  echo Error, missing: PROMETHEUS_JMX_TARGETS="'10.5.0.11:7070','10.5.0.12:7070'"
  exit 1
fi
if [ "$PROMETHEUS_GO_TARGETS" = "" ]; then
  echo Error, missing: PROMETHEUS_GO_TARGETS="'10.5.1.10:9200','10.5.1.10:9205','10.5.0.101:9201','10.5.0.102:9201'"
  exit 1
fi
if [ "$RABBITMQ_ERLANG_FILENAME" = "" ]; then
  echo Error, missing: RABBITMQ_ERLANG_FILENAME=esl-erlang_27.3.4-1_arm64.deb
  exit 1
fi
if [ "$RABBITMQ_SERVER_FILENAME" = "" ]; then
  echo Error, missing: RABBITMQ_SERVER_FILENAME=rabbitmq-server_4.2.0-1_all.deb
  exit 1
fi
if [ "$ACTIVEMQ_CLASSIC_SERVER_FILENAME" = "" ]; then
  echo Error, missing: ACTIVEMQ_CLASSIC_SERVER_FILENAME=apache-activemq-6.1.8-bin.tar.gz
  exit 1
fi
if [ "$ACTIVEMQ_ARTEMIS_SERVER_FILENAME" = "" ]; then
  echo Error, missing: ACTIVEMQ_ARTEMIS_SERVER_FILENAME=apache-artemis-2.44.0-bin.tar.gz
  exit 1
fi

if [ "$INTERNAL_CAPIMQ_BROKER_PORT" = "" ]; then
  echo Error, missing: INTERNAL_CAPIMQ_BROKER_PORT=7654
  exit 1
fi
if [ "$EXTERNAL_CAPIMQ_BROKER_PORT" = "" ]; then
  echo Error, missing: EXTERNAL_CAPIMQ_BROKER_PORT=7655
  exit 1
fi
if [ "$CAPIMQ_BROKER_MAX_MESSAGES" = "" ]; then
  echo Error, missing: CAPIMQ_BROKER_MAX_MESSAGES=10000000
  exit 1
fi
if [ "$CAPIMQ_BROKER_RETURNED_DELIVERY_DELAY" = "" ]; then
  echo Error, missing: CAPIMQ_BROKER_RETURNED_DELIVERY_DELAY=500
  exit 1
fi
if [ "$CAPIMQ_BROKER_DEAD_AFTER_NO_HEARTBEAT_TIMEOUT" = "" ]; then
  echo Error, missing: CAPIMQ_BROKER_DEAD_AFTER_NO_HEARTBEAT_TIMEOUT=10000
  exit 1
fi
if [ "$WEBAPI_PROMETHEUS_EXPORTER_PORT" = "" ]; then
  echo Error, missing: WEBAPI_PROMETHEUS_EXPORTER_PORT=9200
  exit 1
fi
if [ "$CAPIMQ_PROMETHEUS_EXPORTER_PORT" = "" ]; then
  echo Error, missing: CAPIMQ_PROMETHEUS_EXPORTER_PORT=9205
  exit 1
fi

if [ "$BASTION_EXTERNAL_IP_ADDRESS" = "" ]; then
  echo Error, missing: BASTION_EXTERNAL_IP_ADDRESS=...
  exit 1
fi

# Use $SSH_USER
if [ ! -d /home/$SSH_USER ]; then
  mkdir -p /home/$SSH_USER
fi
sudo chmod 755 /home/$SSH_USER

if [ ! -d /home/$SSH_USER/bin ]; then
  mkdir -p /home/$SSH_USER/bin
fi
sudo chmod 755 /home/$SSH_USER/bin

sudo mkdir /var/log/capillaries
sudo chown $SSH_USER /var/log/capillaries


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
sed -i -e 's~localhost:6543~'$BASTION_EXTERNAL_IP_ADDRESS':'$EXTERNAL_WEBAPI_PORT'~g' /home/$SSH_USER/ui/_app/immutable/entry/*.js



# Install Webapi



CAPI_BINARY=capiwebapi

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

# ActiveMQ reverse proxy

ACTIVEMQ_CONFIG_FILE=/etc/nginx/sites-available/activemq
if [ -f "$ACTIVEMQ_CONFIG_FILE" ]; then
  sudo rm -f $ACTIVEMQ_CONFIG_FILE
fi

sudo tee $ACTIVEMQ_CONFIG_FILE <<EOF
server {
    listen $EXTERNAL_ACTIVEMQ_CONSOLE_PORT;
    location / {
        proxy_pass http://localhost:8161;
        include proxy_params;
        include includes/allowed_ips.conf;
    }
}
EOF

if [ ! -L "/etc/nginx/sites-enabled/activemq" ]; then
  sudo ln -s $ACTIVEMQ_CONFIG_FILE /etc/nginx/sites-enabled/
fi

sudo nginx -t
if [ "$?" -ne "0" ]; then
    echo nginx activemq reverse proxy config error, exiting
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

# capimq reverse proxy
CAPIMQ_CONFIG_FILE=/etc/nginx/sites-available/capimq
if [ -f "$CAPIMQ_CONFIG_FILE" ]; then
  sudo rm -f $CAPIMQ_CONFIG_FILE
fi

sudo tee $CAPIMQ_CONFIG_FILE <<EOF
server {
    listen $EXTERNAL_CAPIMQ_BROKER_PORT;
    location / {
        proxy_pass http://localhost:$INTERNAL_CAPIMQ_BROKER_PORT;
        include proxy_params;
        include includes/allowed_ips.conf;
    }
}
EOF

if [ ! -L "/etc/nginx/sites-enabled/capimq" ]; then
  sudo ln -s $CAPIMQ_CONFIG_FILE /etc/nginx/sites-enabled/
fi

sudo nginx -t
if [ "$?" -ne "0" ]; then
    echo nginx capimq reverse proxy config error, exiting
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

# Add hostname to the log file names and send them to S3 every 1 min (crontab)
SEND_LOGS_FILE=/home/$SSH_USER/sendlogs.sh
sudo tee $SEND_LOGS_FILE <<EOF
#!/bin/bash
# Send SIGHUP to the running binary, it will rotate the log using Lumberjack
if [ -s /var/log/capillaries/capiwebapi.log ]; then
  ps axf | grep capiwebapi | grep -v grep | awk '{print "kill -s 1 " \$1}' | sh
fi
if [ -s /var/log/capillaries/capimq.log ]; then
  ps axf | grep capimq | grep -v grep | awk '{print "kill -s 1 " \$1}' | sh
fi
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
EOF
sudo chmod 744 $SEND_LOGS_FILE
sudo su $SSH_USER -c "echo \"*/1 * * * * $SEND_LOGS_FILE\" | crontab -"


# Everything in ~ should belong to ssh user
sudo chown -R $SSH_USER /home/$SSH_USER


# Run Webapi

# If we ever use https and/or domain names, or use other port than 80, revisit this piece.
# AWS region is required because S3 bucket pointer is a URI, not a URL
echo Running webapi with GOMEMLIMIT="$WEBAPI_GOMEMLIMIT_GB"GiB GOGC=$WEBAPI_GOGC AWS_DEFAULT_REGION=$AWSREGION CAPI_PROMETHEUS_EXPORTER_PORT=$WEBAPI_PROMETHEUS_EXPORTER_PORT CAPI_WEBAPI_ACCESS_CONTROL_ALLOW_ORIGIN="http://$BASTION_EXTERNAL_IP_ADDRESS" CAPI_WEBAPI_PORT=$INTERNAL_WEBAPI_PORT CAPI_CASSANDRA_HOSTS="$CASSANDRA_HOSTS" CAPI_CASSANDRA_PORT=$CASSANDRA_PORT CAPI_CASSANDRA_USERNAME="$CASSANDRA_USERNAME" CAPI_CASSANDRA_PASSWORD="$CASSANDRA_PASSWORD" CAPI_CASSANDRA_ENABLE_HOST_VERIFICATION=false CAPI_CASSANDRA_KEYSPACE_REPLICATION_CONFIG="{ 'class' : 'NetworkTopologyStrategy', 'datacenter1' : 1 }" CAPI_CASSANDRA_CONSISTENCY=LOCAL_QUORUM CAPI_MQ_TYPE=$MQ_TYPE CAPI_CAPIMQ_CLIENT_URL=$CAPIMQ_CLIENT_URL CAPI_AMQP10_URL="$AMQP10_URL" CAPI_AMQP10_ADDRESS="$AMQP10_ADDRESS" CAPI_CASSANDRA_TIMEOUT=15000 CAPI_LOG_LEVEL=info CAPI_LOG_FILE="/var/log/capillaries/capiwebapi.log"
echo To stop it: 'kill -9 $(ps aux |grep capiwebapi | grep bin | awk '"'"'{print $2}'"'"')'
GOMEMLIMIT="$WEBAPI_GOMEMLIMIT_GB"GiB GOGC=$WEBAPI_GOGC AWS_DEFAULT_REGION=$AWSREGION CAPI_PROMETHEUS_EXPORTER_PORT=$WEBAPI_PROMETHEUS_EXPORTER_PORT CAPI_WEBAPI_ACCESS_CONTROL_ALLOW_ORIGIN="http://$BASTION_EXTERNAL_IP_ADDRESS" CAPI_WEBAPI_PORT=$INTERNAL_WEBAPI_PORT CAPI_CASSANDRA_HOSTS="$CASSANDRA_HOSTS" CAPI_CASSANDRA_PORT=$CASSANDRA_PORT CAPI_CASSANDRA_USERNAME="$CASSANDRA_USERNAME" CAPI_CASSANDRA_PASSWORD="$CASSANDRA_PASSWORD" CAPI_CASSANDRA_ENABLE_HOST_VERIFICATION=false CAPI_CASSANDRA_KEYSPACE_REPLICATION_CONFIG="{ 'class' : 'NetworkTopologyStrategy', 'datacenter1' : 1 }" CAPI_CASSANDRA_CONSISTENCY=LOCAL_QUORUM CAPI_MQ_TYPE=$MQ_TYPE CAPI_CAPIMQ_CLIENT_URL=$CAPIMQ_CLIENT_URL CAPI_AMQP10_URL="$AMQP10_URL" CAPI_AMQP10_ADDRESS="$AMQP10_ADDRESS" CAPI_CASSANDRA_TIMEOUT=15000 CAPI_LOG_LEVEL=info CAPI_LOG_FILE="/var/log/capillaries/capiwebapi.log" /home/$SSH_USER/bin/capiwebapi &>/dev/null &





# Install AMQP10 server

cd /home/$SSH_USER



if [ "$MQ_TYPE" = "capimq" ]; then

CAPI_BINARY=capimq

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

echo Running capimq with CAPI_CAPIMQ_BROKER_ACCESS_CONTROL_ALLOW_ORIGIN=none CAPI_CAPIMQ_BROKER_PORT=$INTERNAL_CAPIMQ_PORT CAPI_CAPIMQ_BROKER_RETURNED_DELIVERY_DELAY=$CAPIMQ_BROKER_RETURNED_DELIVERY_DELAY CAPI_CAPIMQ_BROKER_MAX_MESSAGES=$CAPIMQ_BROKER_MAX_MESSAGES CAPI_CAPIMQ_BROKER_DEAD_AFTER_NO_HEARTBEAT_TIMEOUT=$CAPIMQ_BROKER_DEAD_AFTER_NO_HEARTBEAT_TIMEOUT CAPI_PROMETHEUS_EXPORTER_PORT=$CAPIMQ_PROMETHEUS_EXPORTER_PORT CAPI_LOG_LEVEL=info CAPI_LOG_FILE="/var/log/capillaries/capimq.log"
echo To stop it: 'kill -9 $(ps aux |grep capimq | grep bin | awk '"'"'{print $2}'"'"')'
CAPI_CAPIMQ_BROKER_ACCESS_CONTROL_ALLOW_ORIGIN=none CAPI_CAPIMQ_BROKER_PORT=$INTERNAL_CAPIMQ_BROKER_PORT CAPI_CAPIMQ_BROKER_RETURNED_DELIVERY_DELAY=$CAPIMQ_BROKER_RETURNED_DELIVERY_DELAY CAPI_CAPIMQ_BROKER_MAX_MESSAGES=$CAPIMQ_BROKER_MAX_MESSAGES CAPI_CAPIMQ_BROKER_DEAD_AFTER_NO_HEARTBEAT_TIMEOUT=$CAPIMQ_BROKER_DEAD_AFTER_NO_HEARTBEAT_TIMEOUT CAPI_PROMETHEUS_EXPORTER_PORT=$CAPIMQ_PROMETHEUS_EXPORTER_PORT CAPI_LOG_LEVEL=info CAPI_LOG_FILE="/var/log/capillaries/capimq.log" /home/$SSH_USER/bin/capimq &>/dev/null &

elif [ "$AMQP10_SERVER_FLAVOR" = "activemq-classic" ]; then

sudo DEBIAN_FRONTEND=noninteractive apt-get install -y openjdk-17-jdk
if [ "$?" -ne "0" ]; then
    echo openjdk install error, exiting
    exit $?
fi

curl -LOs $CAPILLARIES_RELEASE_URL/$ACTIVEMQ_CLASSIC_SERVER_FILENAME
if [ "$?" -ne "0" ]; then
    echo activemq download error, exiting
    exit $?
fi

sudo tar -xzf $ACTIVEMQ_CLASSIC_SERVER_FILENAME
ACTIVEMQ_CLASSIC_SERVER_DIR=$(basename $ACTIVEMQ_CLASSIC_SERVER_FILENAME "-bin.tar.gz")

sudo mkdir /opt/activemq
sudo mv $ACTIVEMQ_CLASSIC_SERVER_DIR/* /opt/activemq

sudo addgroup --system activemq
sudo adduser --system --ingroup activemq --no-create-home --disabled-password activemq
sudo chown -R activemq:activemq /opt/activemq


ACTIVEMQ_SERVICE_FILE=/etc/systemd/system/activemq.service
sudo rm -f $ACTIVEMQ_SERVICE_FILE
sudo tee $ACTIVEMQ_SERVICE_FILE <<EOF
[Unit]
Description=Apache ActiveMQ
After=network.target
[Service]
Type=forking
User=activemq
Group=activemq
ExecStart=/opt/activemq/bin/activemq start
ExecStop=/opt/activemq/bin/activemq stop
[Install]
WantedBy=multi-user.target
EOF

sudo sed -i -e 's~<property name="host" value="127.0.0.1"/>~<property name="host" value="0.0.0.0"/>~g' /opt/activemq/conf/jetty.xml

sudo systemctl daemon-reload
sudo systemctl enable activemq
sudo systemctl start activemq

ACTIVEMQXML_FILE=/opt/activemq/conf/activemq.xml
sudo rm -f $ACTIVEMQXML_FILE
sudo tee $ACTIVEMQXML_FILE <<EOF
<beans xmlns="http://www.springframework.org/schema/beans"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.springframework.org/schema/beans http://www.springframework.org/schema/beans/spring-beans.xsd
            http://activemq.apache.org/schema/core http://activemq.apache.org/schema/core/activemq-core.xsd">
  <bean class="org.springframework.beans.factory.config.PropertyPlaceholderConfigurer">
    <property name="locations">
      <value>file:\${activemq.conf}/credentials.properties</value>
    </property>
  </bean>
  <broker xmlns="http://activemq.apache.org/schema/core" brokerName="localhost" dataDirectory="\${activemq.data}" schedulerSupport="true">
    <destinationPolicy>
      <policyMap>
        <policyEntries>
          <policyEntry topic=">">
            <pendingMessageLimitStrategy>
              <constantPendingMessageLimitStrategy limit="1000" />
            </pendingMessageLimitStrategy>
          </policyEntry>
        </policyEntries>
      </policyMap>
    </destinationPolicy>
    <managementContext>
      <managementContext createConnector="false" />
    </managementContext>
    <persistenceAdapter>
      <kahaDB directory="\${activemq.data}/kahadb" />
    </persistenceAdapter>
    <systemUsage>
      <systemUsage>
        <memoryUsage>
          <memoryUsage percentOfJvmHeap="70" />
        </memoryUsage>
        <storeUsage>
          <storeUsage limit="100 gb" />
        </storeUsage>
        <tempUsage>
          <tempUsage limit="50 gb" />
        </tempUsage>
      </systemUsage>
    </systemUsage>
    <transportConnectors>
      <transportConnector name="openwire" uri="tcp://0.0.0.0:61616?maximumConnections=1000&amp;wireFormat.maxFrameSize=104857600" />
      <transportConnector name="amqp" uri="amqp://0.0.0.0:5672?maximumConnections=1000&amp;wireFormat.maxFrameSize=104857600" />
      <transportConnector name="stomp" uri="stomp://0.0.0.0:61613?maximumConnections=1000&amp;wireFormat.maxFrameSize=104857600" />
      <transportConnector name="mqtt" uri="mqtt://0.0.0.0:1883?maximumConnections=1000&amp;wireFormat.maxFrameSize=104857600" />
      <transportConnector name="ws" uri="ws://0.0.0.0:61614?maximumConnections=1000&amp;wireFormat.maxFrameSize=104857600" />
    </transportConnectors>
    <shutdownHooks>
      <bean xmlns="http://www.springframework.org/schema/beans" class="org.apache.activemq.hooks.SpringContextHook" />
    </shutdownHooks>
    <plugins>
      <!-- added -->
      <redeliveryPlugin>
        <redeliveryPolicyMap>
          <redeliveryPolicyMap>
            <redeliveryPolicyEntries>
              <redeliveryPolicy queue="capillaries" maximumRedeliveries="-1" initialRedeliveryDelay="5000" redeliveryDelay="5000" />
            </redeliveryPolicyEntries>
            <defaultEntry>
              <redeliveryPolicy maximumRedeliveries="-1" initialRedeliveryDelay="5000" redeliveryDelay="5000" />
            </defaultEntry>
          </redeliveryPolicyMap>
        </redeliveryPolicyMap>
      </redeliveryPlugin>
      <simpleAuthenticationPlugin anonymousAccessAllowed="false">
        <users>
          <authenticationUser username="$AMQP10_USER_NAME" password="$AMQP10_USER_PASS" groups="users"/>
        </users>
      </simpleAuthenticationPlugin>
    </plugins>
  </broker>
  <import resource="jetty.xml" />
</beans>
EOF

sudo tee /opt/activemq/conf/users.properties <<EOF
$AMQP10_ADMIN_NAME=$AMQP10_ADMIN_PASS
$AMQP10_USER_NAME=$AMQP10_USER_PASS
EOF

sudo tee /opt/activemq/conf/groups.properties <<EOF
admins=$AMQP10_ADMIN_NAME
users=$AMQP10_USER_NAME
EOF

sudo systemctl stop activemq
sudo systemctl start activemq

elif [ "$AMQP10_SERVER_FLAVOR" = "activemq-artemis" ]; then

sudo DEBIAN_FRONTEND=noninteractive apt-get install -y openjdk-21-jdk
if [ "$?" -ne "0" ]; then
    echo openjdk install error, exiting
    exit $?
fi

curl -LOs $CAPILLARIES_RELEASE_URL/$ACTIVEMQ_ARTEMIS_SERVER_FILENAME
if [ "$?" -ne "0" ]; then
    echo activemq download error, exiting
    exit $?
fi

sudo tar -xzf $ACTIVEMQ_ARTEMIS_SERVER_FILENAME -C /opt/
ACTIVEMQ_ARTEMIS_SERVER_DIR=$(basename $ACTIVEMQ_ARTEMIS_SERVER_FILENAME "-bin.tar.gz")

sudo mv /opt/$ACTIVEMQ_ARTEMIS_SERVER_DIR /opt/activemq-artemis

sudo addgroup --system activemq
sudo adduser --system --ingroup activemq --no-create-home --disabled-password activemq
sudo chown -R activemq:activemq /opt/activemq-artemis

# Create broker instance in /var/lib/activemq-artemis-broker
cd /opt/activemq-artemis/bin
sudo mkdir /var/lib/activemq-artemis-broker
sudo chown -R activemq:activemq /var/lib/activemq-artemis-broker
sudo -u activemq ./artemis create /var/lib/activemq-artemis-broker --user $AMQP10_ADMIN_NAME --password $AMQP10_ADMIN_PASS --allow-anonymous --relax-jolokia

# TODO: add JMX exporter for ActiveMQ

ACTIVEMQ_SERVICE_FILE=/etc/systemd/system/activemq-artemis.service
sudo rm -f $ACTIVEMQ_SERVICE_FILE
sudo tee $ACTIVEMQ_SERVICE_FILE <<EOF
[Unit]
Description=ActiveMQ Artemis Broker
After=network.target
[Service]
Type=forking
User=activemq
Group=activemq
ExecStart=/var/lib/activemq-artemis-broker/bin/artemis-service start
ExecStop=/var/lib/activemq-artemis-broker/bin/artemis-service stop
Restart=on-failure
[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable activemq-artemis
sudo systemctl start activemq-artemis

ACTIVEMQ_BROKERXML_FILE=/var/lib/activemq-artemis-broker/etc/broker.xml
sudo rm -f $ACTIVEMQ_BROKERXML_FILE
sudo tee $ACTIVEMQ_BROKERXML_FILE <<EOF
<?xml version='1.0'?>
<configuration xmlns="urn:activemq"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xmlns:xi="http://www.w3.org/2001/XInclude" xsi:schemaLocation="urn:activemq /schema/artemis-configuration.xsd">
  <core xmlns="urn:activemq:core"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:activemq:core ">
    <name>0.0.0.0</name>
    <persistence-enabled>true</persistence-enabled>
    <max-redelivery-records>1</max-redelivery-records>
    <journal-type>NIO</journal-type>
    <purge-page-folders>false</purge-page-folders>
    <paging-directory>data/paging</paging-directory>
    <bindings-directory>data/bindings</bindings-directory>
    <journal-directory>data/journal</journal-directory>
    <large-messages-directory>data/large-messages</large-messages-directory>
    <journal-retention-directory period="7" unit="DAYS" storage-limit="10G">data/retention</journal-retention-directory>
    <journal-datasync>true</journal-datasync>
    <journal-min-files>2</journal-min-files>
    <journal-pool-files>10</journal-pool-files>
    <journal-device-block-size>4096</journal-device-block-size>
    <journal-file-size>10M</journal-file-size>
    <journal-buffer-timeout>344000</journal-buffer-timeout>
    <journal-max-io>1</journal-max-io>
    <disk-scan-period>5000</disk-scan-period>
    <max-disk-usage>90</max-disk-usage>
    <critical-analyzer>true</critical-analyzer>
    <critical-analyzer-timeout>120000</critical-analyzer-timeout>
    <critical-analyzer-check-period>60000</critical-analyzer-check-period>
    <critical-analyzer-policy>HALT</critical-analyzer-policy>
    <page-sync-timeout>344000</page-sync-timeout>
    <global-max-messages>-1</global-max-messages>
    <acceptors>
      <acceptor name="artemis"> tcp://0.0.0.0:61616?tcpSendBufferSize=1048576;tcpReceiveBufferSize=1048576;amqpMinLargeMessageSize=102400;protocols=CORE,AMQP,STOMP,HORNETQ,MQTT,OPENWIRE;useEpoll=true;amqpCredits=1000;amqpLowCredits=300;amqpDuplicateDetection=true;supportAdvisory=false;suppressInternalManagementObjects=false</acceptor>
      <acceptor name="amqp"> tcp://0.0.0.0:5672?tcpSendBufferSize=1048576;tcpReceiveBufferSize=1048576;protocols=AMQP;useEpoll=true;amqpCredits=1000;amqpLowCredits=300;amqpMinLargeMessageSize=102400;amqpDuplicateDetection=true</acceptor>
      <acceptor name="stomp"> tcp://0.0.0.0:61613?tcpSendBufferSize=1048576;tcpReceiveBufferSize=1048576;protocols=STOMP;useEpoll=true</acceptor>
      <acceptor name="hornetq"> tcp://0.0.0.0:5445?anycastPrefix=jms.queue.;multicastPrefix=jms.topic.;protocols=HORNETQ,STOMP;useEpoll=true</acceptor>
      <acceptor name="mqtt"> tcp://0.0.0.0:1883?tcpSendBufferSize=1048576;tcpReceiveBufferSize=1048576;protocols=MQTT;useEpoll=true</acceptor>
    </acceptors>
    <security-settings>
      <security-setting match="#">
        <permission type="createNonDurableQueue" roles="amq" />
        <permission type="deleteNonDurableQueue" roles="amq" />
        <permission type="createDurableQueue" roles="amq" />
        <permission type="deleteDurableQueue" roles="amq" />
        <permission type="createAddress" roles="amq" />
        <permission type="deleteAddress" roles="amq" />
        <permission type="consume" roles="amq" />
        <permission type="browse" roles="amq" />
        <permission type="send" roles="amq" />
        <permission type="manage" roles="amq" />
      </security-setting>
    </security-settings>
    <address-settings>
      <!-- added, we want to mimic: -->
      <!-- /opt/activemq-artemis/bin/artemis queue create &#45&#45name capillaries &#45&#45address capillaries &#45&#45auto-create-address &#45&#45anycast &#45&#45user artemis &#45&#45password artemis &#45&#45no-durable &#45&#45preserve-on-no-consumers -->
      <!-- Do not involve DLQ, see max-delivery-attempts=-1 and dead-letter-address="" -->
      <address-setting match="capillaries">
        <auto-create-addresses>true</auto-create-addresses>
        <default-address-routing-type>ANYCAST</default-address-routing-type>
        <management-message-attribute-size-limit>1024</management-message-attribute-size-limit>
        <default-purge-on-no-consumers>false</default-purge-on-no-consumers>
        <redelivery-delay>5000</redelivery-delay>
        <redelivery-delay-multiplier>1.0</redelivery-delay-multiplier>
        <max-redelivery-delay>5000</max-redelivery-delay>
        <max-delivery-attempts>-1</max-delivery-attempts>
      </address-setting>
      <address-setting match="activemq.management.#">
        <dead-letter-address>DLQ</dead-letter-address>
        <expiry-address>ExpiryQueue</expiry-address>
        <redelivery-delay>0</redelivery-delay>
        <max-size-bytes>-1</max-size-bytes>
        <message-counter-history-day-limit>10</message-counter-history-day-limit>
        <address-full-policy>PAGE</address-full-policy>
        <auto-create-queues>true</auto-create-queues>
        <auto-create-addresses>true</auto-create-addresses>
      </address-setting>
      <address-setting match="#">
        <dead-letter-address>DLQ</dead-letter-address>
        <expiry-address>ExpiryQueue</expiry-address>
        <redelivery-delay>0</redelivery-delay>
        <message-counter-history-day-limit>10</message-counter-history-day-limit>
        <address-full-policy>PAGE</address-full-policy>
        <auto-create-queues>true</auto-create-queues>
        <auto-create-addresses>true</auto-create-addresses>
        <auto-delete-queues>false</auto-delete-queues>
        <auto-delete-addresses>false</auto-delete-addresses>
        <page-size-bytes>10M</page-size-bytes>
        <max-size-bytes>-1</max-size-bytes>
        <max-size-messages>-1</max-size-messages>
        <max-read-page-messages>-1</max-read-page-messages>
        <max-read-page-bytes>20M</max-read-page-bytes>
        <page-limit-bytes>-1</page-limit-bytes>
        <page-limit-messages>-1</page-limit-messages>
      </address-setting>
    </address-settings>
    <addresses>
      <address name="DLQ">
        <anycast>
          <queue name="DLQ" />
        </anycast>
      </address>
      <address name="ExpiryQueue">
        <anycast>
          <queue name="ExpiryQueue" />
        </anycast>
      </address>
      <!-- added, we want to mimic: -->
      <!-- /opt/activemq-artemis/bin/artemis queue create &#45&#45name capillaries &#45&#45address capillaries &#45&#45auto-create-address &#45&#45anycast &#45&#45user artemis &#45&#45password artemis &#45&#45no-durable &#45&#45preserve-on-no-consumers -->
      <address name="capillaries">
        <anycast>
          <queue name="capillaries">
            <durable>false</durable>
          </queue>
        </anycast>
      </address>
    </addresses>
  </core>
</configuration>
EOF

sudo tee /var/lib/activemq-artemis-broker/etc/artemis-users.properties <<EOF
$AMQP10_ADMIN_NAME=$AMQP10_ADMIN_PASS
$AMQP10_USER_NAME=$AMQP10_USER_PASS
EOF

sudo tee /var/lib/activemq-artemis-broker/etc/artemis-roles.properties <<EOF
amq=$AMQP10_ADMIN_NAME,$AMQP10_USER_NAME
EOF

# Not needed, but just in case
sudo systemctl restart activemq-artemis

elif [ "$AMQP10_SERVER_FLAVOR" = "rabbitmq" ]; then

curl -LOs $CAPILLARIES_RELEASE_URL/$RABBITMQ_ERLANG_FILENAME
sudo DEBIAN_FRONTEND=noninteractive apt install -y ./$RABBITMQ_ERLANG_FILENAME
if [ "$?" -ne "0" ]; then
    echo erlang install error, exiting
    exit $?
fi

curl -LOs $CAPILLARIES_RELEASE_URL/$RABBITMQ_SERVER_FILENAME
sudo DEBIAN_FRONTEND=noninteractive apt install -y ./$RABBITMQ_SERVER_FILENAME
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
sudo rabbitmqctl add_user $AMQP10_ADMIN_NAME $AMQP10_ADMIN_PASS
sudo rabbitmqctl set_user_tags $AMQP10_ADMIN_NAME administrator
sudo rabbitmqctl set_permissions -p / $AMQP10_ADMIN_NAME ".*" ".*" ".*"

# Delete default guest user
sudo rabbitmqctl list_users
sudo rabbitmqctl delete_user guest

# Capillaries daemon and webapi use this account
sudo rabbitmqctl add_user $AMQP10_USER_NAME $AMQP10_USER_PASS
sudo rabbitmqctl set_permissions -p / $AMQP10_USER_NAME ".*" ".*" ".*"

# Install rabbitmqadmin v1 (Python script)
sudo rabbitmq-plugins enable rabbitmq_management
sudo curl -Lo /usr/local/bin/rabbitmqadmin https://raw.githubusercontent.com/rabbitmq/rabbitmq-management/v3.7.8/bin/rabbitmqadmin
sudo sed -i -e "s~env python~env python3~g" /usr/local/bin/rabbitmqadmin
sudo chmod 777 /usr/local/bin/rabbitmqadmin

# The following commands set up RabbitMQ dead letter queue infrastructure (no ActiveMQ-style redelivery-delay shortcut in RabbitMQ). Very fragile.
# The only parameter you may want to play with is x-message-ttl.
# Please note that this setup assumes that Capillaries Amqp10.address setting is set to "/queues/capidaemon" 
# Some AMQP 1.0 details (like, why "/queues/capidaemon" and not just "my_simple_capillaries_queue") at https://www.rabbitmq.com/docs/amqp
# Dead letter exchange circle of life explained at https://www.cloudamqp.com/blog/when-and-how-to-use-the-rabbitmq-dead-letter-exchange.html
sudo rabbitmqadmin declare exchange --vhost=/ name=capillaries type=direct durable=false -u capiadmin -p capiadminpass
sudo rabbitmqadmin declare exchange --vhost=/ name=capillaries.dlx type=direct durable=false -u capiadmin -p capiadminpass
sudo rabbitmqadmin declare queue --vhost=/ name=capidaemon durable=false arguments='{"x-dead-letter-exchange":"capillaries.dlx", "x-dead-letter-routing-key":"capidaemon.dlq"}' -u capiadmin -p capiadminpass
sudo rabbitmqadmin declare queue --vhost=/ name=capidaemon.dlq durable=false arguments='{"x-message-ttl":5000, "x-dead-letter-exchange":"capillaries", "x-dead-letter-routing-key":"capidaemon"}' -u capiadmin -p capiadminpass
sudo rabbitmqadmin --vhost=/ declare binding source="capillaries" destination_type="queue" destination="capidaemon" routing_key="capidaemon" -u capiadmin -p capiadminpass
sudo rabbitmqadmin --vhost=/ declare binding source="capillaries.dlx" destination_type="queue" destination="capidaemon.dlq" routing_key="capidaemon.dlq" -u capiadmin -p capiadminpass

curl -s http://localhost:15672
if [ "$?" -ne "0" ]; then
    echo Cannot check localhost:15672
    exit $?
fi

else 
  echo Invalid AMQP10_SERVER_FLAVOR: $AMQP10_SERVER_FLAVOR
  exit 1
fi




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





# Install Prometheus server





# Create users
sudo useradd --no-create-home --shell /bin/false prometheus

# Before we download the Prometheus binaries, create the necessary directories for storing Prometheus’ files and data. Following standard Linux conventions, we’ll create a directory in /etc for Prometheus’ configuration files and a directory in /var/lib for its data.
sudo mkdir /etc/prometheus
sudo mkdir /var/lib/prometheus

# Now, set the user and group ownership on the new directories to the prometheus user.
sudo chown prometheus:prometheus /etc/prometheus
sudo chown prometheus:prometheus /var/lib/prometheus


# Downloading Prometheus

cd /home/$SSH_USER
curl -LOs $CAPILLARIES_RELEASE_URL/$PROMETHEUS_SERVER_FILENAME
if [ "$?" -ne "0" ]; then
    echo Cannot download, exiting
    exit $?
fi
tar xvf $PROMETHEUS_SERVER_FILENAME
PROMETHEUS_SERVER_DIR=$(basename $PROMETHEUS_SERVER_FILENAME .tar.gz)

# Copy the two binaries to the /usr/local/bin directory.

sudo cp $PROMETHEUS_SERVER_DIR/prometheus /usr/local/bin/
sudo cp $PROMETHEUS_SERVER_DIR/promtool /usr/local/bin/

# Set the user and group ownership on the binaries to the prometheus user created in Step 1.
sudo chown prometheus:prometheus /usr/local/bin/prometheus
sudo chown prometheus:prometheus /usr/local/bin/promtool

rm -fR $PROMETHEUS_SERVER_FILENAME $PROMETHEUS_SERVER_DIR




# Configure Prometheus server



# Prometheus server (assuming node exporter also running on it)
# https://www.digitalocean.com/community/tutorials/how-to-install-prometheus-on-ubuntu-16-04

#sudo systemctl stop prometheus

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
