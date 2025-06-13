#!/bin/bash

echo Running bastion.sh.tpl in $(pwd)

sudo DEBIAN_FRONTEND=noninteractive apt-get update -y

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


# Use ${ssh_user}

if [ ! -d /home/${ssh_user} ]; then
  mkdir -p /home/${ssh_user}
fi
sudo chmod 755 /home/${ssh_user}


# Install UI

if [ ! -d /home/${ssh_user}/ui ]; then
  mkdir -p /home/${ssh_user}/ui
fi
sudo chmod 755 /home/${ssh_user}/ui
cd /home/${ssh_user}/ui

curl -LOs ${capillaries_release_url}/webui/webui.tgz
if [ "$?" -ne "0" ]; then
    echo "Cannot download webui from ${capillaries_release_url}/webui/webui.tgz to /home/${ssh_user}/ui"
    exit $?
fi

tar xvzf webui.tgz
rm webui.tgz

# Tweak UI so it calls the proper capiwebapi URL
# This is not idempotent. It's actually pretty hacky.
echo Patching WebUI to use external Webapi ip:port ${bastion_external_ip_address}:${external_webapi_port}
sed -i -e 's~localhost:6543~'${bastion_external_ip_address}':'${external_webapi_port}'~g' /home/${ssh_user}/ui/_app/immutable/chunks/*.js

UI_CONFIG_FILE=/etc/nginx/sites-available/ui
if [ -f "$UI_CONFIG_FILE" ]; then
  sudo rm -f $UI_CONFIG_FILE
fi

sudo tee $UI_CONFIG_FILE <<EOF
server {
  listen 80;
  listen [::]:80;
  root /home/${ssh_user}/ui;
  index index.html;
  location / {
    include includes/allowed_ips.conf;
  }
}
EOF

if [ ! -L "/etc/nginx/sites-enabled/ui" ]; then
  sudo ln -s $UI_CONFIG_FILE /etc/nginx/sites-enabled/
fi


# Install Webapi

CAPI_BINARY=capiwebapi

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
sudo chmod 744 $CAPI_BINARY

sudo mkdir /var/log/capillaries
sudo chown ${ssh_user} /var/log/capillaries

# Reverse proxies

# Webapi reverse proxy
WEBAPI_CONFIG_FILE=/etc/nginx/sites-available/webapi
if [ -f "$WEBAPI_CONFIG_FILE" ]; then
  sudo rm -f $WEBAPI_CONFIG_FILE
fi

sudo tee $WEBAPI_CONFIG_FILE <<EOF
server {
    listen ${external_webapi_port};
    location / {
        proxy_pass http://localhost:${internal_webapi_port};
        include proxy_params;
        include includes/allowed_ips.conf;
    }
}
EOF

if [ ! -L "/etc/nginx/sites-enabled/webapi" ]; then
  sudo ln -s $WEBAPI_CONFIG_FILE /etc/nginx/sites-enabled/
fi


# Config IP address whitelist

if [ ! -d "/etc/nginx/includes" ]; then
  sudo mkdir /etc/nginx/includes
fi

WHITELIST_CONFIG_FILE=/etc/nginx/includes/allowed_ips.conf

if [ -f "$WHITELIST_CONFIG_FILE" ]; then
  sudo rm $WHITELIST_CONFIG_FILE
fi
sudo touch $WHITELIST_CONFIG_FILE

IFS=',' read -ra CIDR <<< "${bastion_allowed_ips}"
for i in "$${CIDR[@]}"; do
  echo "allow $i;" | sudo tee -a $WHITELIST_CONFIG_FILE
done
echo "deny all;" | sudo tee -a $WHITELIST_CONFIG_FILE


# Restart nginx to pick up changes

sudo nginx -t
if [ "$?" -ne "0" ]; then
    echo nginx config error, exiting
    exit $?
fi

sudo systemctl restart nginx


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
if [ -s /var/log/capillaries/capiwebapi.log ]; then
  # Send SIGHUP to the running binary, it will rotate the log using Lumberjack
  ps axf | grep capiwebapi | grep -v grep | awk '{print "kill -s 1 " \$1}' | sh
  for f in /var/log/capillaries/*.gz;do
    if [ -e \$f ]; then
      # Lumberjack produces: capiwebapi-2025-05-03T21-37-01.283.log.gz
      # Add hostname to it: capiwebapi-2025-05-03T21-37-01.283.ip-10-5-1-10.log.gz
      fname=\$(basename -- "\$f")
      fnamedatetime=\$(echo \$fname|cut -d'.' -f1)
      fnamemillis=\$(echo \$fname|cut -d'.' -f2)
      newfilepath=/var/log/capillaries/\$fnamedatetime.\$fnamemillis.\$HOSTNAME.log.gz
      mv \$f \$newfilepath
      aws s3 cp \$newfilepath ${s3_log_url}/
      rm \$newfilepath
    fi
  done
fi
EOF
sudo chmod 744 $SEND_LOGS_FILE
sudo su ${ssh_user} -c "echo \"*/5 * * * * $SEND_LOGS_FILE\" | crontab -"


# Everything in ~ should belong to ssh user
sudo chown -R ${ssh_user} /home/${ssh_user}


# Run Webapi

# If we ever use https and/or domain names, or use other port than 80, revisit this piece.
# AWS region is required because S3 bucket pointer is a URI, not a URL
sudo su ${ssh_user} -c 'AWS_DEFAULT_REGION=${awsregion} CAPI_WEBAPI_ACCESS_CONTROL_ALLOW_ORIGIN="http://${bastion_external_ip_address}" CAPI_WEBAPI_PORT=${internal_webapi_port} CAPI_CASSANDRA_HOSTS="${cassandra_hosts}" CAPI_CASSANDRA_PORT=${cassandra_port} CAPI_CASSANDRA_USERNAME="${cassandra_username}" CAPI_CASSANDRA_PASSWORD="${cassandra_password}" CAPI_CASSANDRA_CA_PATH="/home/${ssh_user}/.ssh/sf-class2-root.crt" CAPI_CASSANDRA_ENABLE_HOST_VERIFICATION=false CAPI_CASSANDRA_KEYSPACE_REPLICATION_CONFIG="'"{ 'class' : 'SingleRegionStrategy'}"'" CAPI_CASSANDRA_CONSISTENCY=LOCAL_QUORUM CAPI_AMQP091_URL="${rabbitmq_url}" CAPI_CASSANDRA_TIMEOUT=15000 CAPI_LOG_LEVEL=info CAPI_LOG_FILE="/var/log/capillaries/capiwebapi.log" /home/${ssh_user}/bin/capiwebapi &>/dev/null &'