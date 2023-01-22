# https://www.cherryservers.com/blog/how-to-install-and-start-using-rabbitmq-on-ubuntu-22-04

# Expects
# RABBITMQ_ADMIN_NAME=...
# RABBITMQ_ADMIN_PASS=...
# RABBITMQ_USER_NAME=...
# RABBITMQ_USER_PASS=...

sudo apt -y install gnupg apt-transport-https

curl -1sLf "https://keys.openpgp.org/vks/v1/by-fingerprint/0A9AF2115F4687BD29803A206B73A36E6026DFCA" | sudo gpg --dearmor | sudo tee /usr/share/keyrings/com.rabbitmq.team.gpg > /dev/null
curl -1sLf "https://keyserver.ubuntu.com/pks/lookup?op=get&search=0xf77f1eda57ebb1cc" | sudo gpg --dearmor | sudo tee /usr/share/keyrings/net.launchpad.ppa.rabbitmq.erlang.gpg > /dev/null
curl -1sLf "https://packagecloud.io/rabbitmq/rabbitmq-server/gpgkey" | sudo gpg --dearmor | sudo tee /usr/share/keyrings/io.packagecloud.rabbitmq.gpg > /dev/null

# Use RabbitMQ "jammy" release for Ubuntu 22.04:
sudo tee /etc/apt/sources.list.d/rabbitmq.list <<EOF
deb [signed-by=/usr/share/keyrings/net.launchpad.ppa.rabbitmq.erlang.gpg] http://ppa.launchpad.net/rabbitmq/rabbitmq-erlang/ubuntu jammy main
deb-src [signed-by=/usr/share/keyrings/net.launchpad.ppa.rabbitmq.erlang.gpg] http://ppa.launchpad.net/rabbitmq/rabbitmq-erlang/ubuntu jammy main
deb [signed-by=/usr/share/keyrings/io.packagecloud.rabbitmq.gpg] https://packagecloud.io/rabbitmq/rabbitmq-server/ubuntu/ jammy main
deb-src [signed-by=/usr/share/keyrings/io.packagecloud.rabbitmq.gpg] https://packagecloud.io/rabbitmq/rabbitmq-server/ubuntu/ jammy main
EOF

sudo apt -y update

# Erlang
sudo apt -y install erlang-base \
    erlang-asn1 erlang-crypto erlang-eldap erlang-ftp erlang-inets \
    erlang-mnesia erlang-os-mon erlang-parsetools erlang-public-key \
    erlang-runtime-tools erlang-snmp erlang-ssl \
    erlang-syntax-tools erlang-tftp erlang-tools erlang-xmerl

# RabbitMQ server
sudo apt -y --fix-missing install rabbitmq-server

sudo rabbitmq-plugins list

# Enable mgmt console
sudo rabbitmq-plugins enable rabbitmq_management

# Console user mgmt
sudo rabbitmqctl add_user $RABBITMQ_ADMIN_NAME $RABBITMQ_ADMIN_PASS
sudo rabbitmqctl set_user_tags $RABBITMQ_ADMIN_NAME administrator
sudo rabbitmqctl list_users
sudo rabbitmqctl delete_user guest

# Capillaries uses this account
sudo rabbitmqctl add_user $RABBITMQ_USER_NAME $RABBITMQ_USER_PASS
sudo rabbitmqctl set_permissions -p / $RABBITMQ_USER_NAME ".*" ".*" ".*"

curl http://localhost:15672

