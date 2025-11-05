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
if [ "$ACTIVEMQ_URL" = "" ]; then
  echo Error, missing: ACTIVEMQ_URL=amqps://capiuser:capipass@10.5.1.10/
  exit 1
fi
if [ "$ACTIVEMQ_USER_NAME" = "" ]; then
  echo Error, missing: ACTIVEMQ_USER_NAME=capiuser
  exit 1
fi
if [ "$ACTIVEMQ_USER_PASS" = "" ]; then
  echo Error, missing: ACTIVEMQ_USER_PASS=capipass
  exit 1
fi
if [ "$ACTIVEMQ_ADMIN_NAME" = "" ]; then
  echo Error, missing: ACTIVEMQ_ADMIN_NAME=radmin
  exit 1
fi
if [ "$ACTIVEMQ_ADMIN_PASS" = "" ]; then
  echo Error, missing: ACTIVEMQ_ADMIN_PASS=rpass
  exit 1
fi
if [ "$ACTIVEMQ_SERVER_VERSION" = "" ]; then
  echo Error, missing: ACTIVEMQ_SERVER_VERSION=2.43.0
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
sed -i -e 's~localhost:6543~'$BASTION_EXTERNAL_IP_ADDRESS':'$EXTERNAL_WEBAPI_PORT'~g' /home/$SSH_USER/ui/_app/immutable/entry/*.js



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
echo Running webapi with GOMEMLIMIT="$WEBAPI_GOMEMLIMIT_GB"GiB GOGC=$WEBAPI_GOGC AWS_DEFAULT_REGION=$AWSREGION CAPI_PROMETHEUS_EXPORTER_PORT=9200 CAPI_WEBAPI_ACCESS_CONTROL_ALLOW_ORIGIN="http://$BASTION_EXTERNAL_IP_ADDRESS" CAPI_WEBAPI_PORT=$INTERNAL_WEBAPI_PORT CAPI_CASSANDRA_HOSTS="$CASSANDRA_HOSTS" CAPI_CASSANDRA_PORT=$CASSANDRA_PORT CAPI_CASSANDRA_USERNAME="$CASSANDRA_USERNAME" CAPI_CASSANDRA_PASSWORD="$CASSANDRA_PASSWORD" CAPI_CASSANDRA_ENABLE_HOST_VERIFICATION=false CAPI_CASSANDRA_KEYSPACE_REPLICATION_CONFIG="{ 'class' : 'NetworkTopologyStrategy', 'datacenter1' : 1 }" CAPI_CASSANDRA_CONSISTENCY=LOCAL_QUORUM CAPI_AMQP10_URL="$ACTIVEMQ_URL" CAPI_CASSANDRA_TIMEOUT=15000 CAPI_LOG_LEVEL=info CAPI_LOG_FILE="/var/log/capillaries/capiwebapi.log"
echo To stop it: 'kill -9 $(ps aux |grep capiwebapi | grep bin | awk '"'"'{print $2}'"'"')'
GOMEMLIMIT="$WEBAPI_GOMEMLIMIT_GB"GiB GOGC=$WEBAPI_GOGC AWS_DEFAULT_REGION=$AWSREGION CAPI_PROMETHEUS_EXPORTER_PORT=9200 CAPI_WEBAPI_ACCESS_CONTROL_ALLOW_ORIGIN="http://$BASTION_EXTERNAL_IP_ADDRESS" CAPI_WEBAPI_PORT=$INTERNAL_WEBAPI_PORT CAPI_CASSANDRA_HOSTS="$CASSANDRA_HOSTS" CAPI_CASSANDRA_PORT=$CASSANDRA_PORT CAPI_CASSANDRA_USERNAME="$CASSANDRA_USERNAME" CAPI_CASSANDRA_PASSWORD="$CASSANDRA_PASSWORD" CAPI_CASSANDRA_ENABLE_HOST_VERIFICATION=false CAPI_CASSANDRA_KEYSPACE_REPLICATION_CONFIG="{ 'class' : 'NetworkTopologyStrategy', 'datacenter1' : 1 }" CAPI_CASSANDRA_CONSISTENCY=LOCAL_QUORUM CAPI_AMQP10_URL="$ACTIVEMQ_URL" CAPI_CASSANDRA_TIMEOUT=15000 CAPI_LOG_LEVEL=info CAPI_LOG_FILE="/var/log/capillaries/capiwebapi.log" /home/$SSH_USER/bin/capiwebapi &>/dev/null &




# Install ActiveMQ


cd /home/$SSH_USER

sudo DEBIAN_FRONTEND=noninteractive apt-get install -y openjdk-21-jdk
if [ "$?" -ne "0" ]; then
    echo openjdk install error, exiting
    exit $?
fi


curl -LOs  https://archive.apache.org/dist/activemq/activemq-artemis/$ACTIVEMQ_SERVER_VERSION/apache-artemis-$ACTIVEMQ_SERVER_VERSION-bin.tar.gz
if [ "$?" -ne "0" ]; then
    echo activemq download error, exiting
    exit $?
fi

sudo tar -xzf apache-artemis-$ACTIVEMQ_SERVER_VERSION-bin.tar.gz -C /opt/
sudo mv /opt/apache-artemis-$ACTIVEMQ_SERVER_VERSION /opt/activemq-artemis

sudo addgroup --system activemq
sudo adduser --system --ingroup activemq --no-create-home --disabled-password activemq
sudo chown -R activemq:activemq /opt/activemq-artemis

# Create broker instance in /var/lib/activemq-artemis-broker
cd /opt/activemq-artemis/bin
sudo mkdir /var/lib/activemq-artemis-broker
sudo chown -R activemq:activemq /var/lib/activemq-artemis-broker
sudo -u activemq ./artemis create /var/lib/activemq-artemis-broker --user $ACTIVEMQ_ADMIN_NAME --password $ACTIVEMQ_ADMIN_PASS --allow-anonymous --relax-jolokia

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
				<redelivery-collision-avoidance-factor>0.1</redelivery-collision-avoidance-factor>
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
$ACTIVEMQ_ADMIN_NAME=$ACTIVEMQ_ADMIN_PASS
$ACTIVEMQ_USER_NAME=$ACTIVEMQ_USER_PASS
EOF

sudo tee /var/lib/activemq-artemis-broker/etc/artemis-roles.properties <<EOF
amq=$ACTIVEMQ_ADMIN_NAME,$ACTIVEMQ_USER_NAME
EOF

# Not needed, but just in case
sudo systemctl restart activemq-artemis






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
