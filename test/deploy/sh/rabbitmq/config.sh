if [ "$RABBITMQ_ADMIN_NAME" = "" ]; then
  echo Error, missing: RABBITMQ_ADMIN_NAME=...
 exit 1
fi
if [ "$RABBITMQ_ADMIN_PASS" = "" ]; then
  echo Error, missing: RABBITMQ_ADMIN_PASS=...
 exit 1
fi
if [ "$RABBITMQ_USER_NAME" = "" ]; then
  echo Error, missing: RABBITMQ_USER_NAME=...
 exit 1
fi
if [ "$RABBITMQ_ADMIN_PASS" = "" ]; then
  echo Error, missing: RABBITMQ_USER_PASS=...
 exit 1
fi

# Mkae sure it's started
sudo systemctl start rabbitmq-server

# Enable mgmt console
sudo rabbitmq-plugins list
sudo rabbitmq-plugins enable rabbitmq_management

# Console user mgmt
sudo rabbitmqctl add_user $RABBITMQ_ADMIN_NAME $RABBITMQ_ADMIN_PASS
sudo rabbitmqctl set_user_tags $RABBITMQ_ADMIN_NAME administrator
sudo rabbitmqctl set_permissions -p / $RABBITMQ_ADMIN_NAME ".*" ".*" ".*"
sudo rabbitmqctl list_users
sudo rabbitmqctl delete_user guest

# Capillaries uses this account
sudo rabbitmqctl add_user $RABBITMQ_USER_NAME $RABBITMQ_USER_PASS
sudo rabbitmqctl set_permissions -p / $RABBITMQ_USER_NAME ".*" ".*" ".*"

curl http://localhost:15672
if [ "$?" -ne "0" ]; then
    echo Cannot check localhost:15672
    exit $?
fi
