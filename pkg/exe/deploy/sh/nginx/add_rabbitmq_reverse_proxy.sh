# nginx reverse proxy
# https://www.digitalocean.com/community/tutorials/how-to-configure-nginx-as-a-reverse-proxy-on-ubuntu-22-04

# Expecting
# RABBITMQ_IP=10.5.0.5

RABBITMQ_CONFIG_FILE=/etc/nginx/sites-available/rabbitmq
sudo rm -f $RABBITMQ_CONFIG_FILE
sudo touch $RABBITMQ_CONFIG_FILE

echo "server {" | sudo tee -a $RABBITMQ_CONFIG_FILE
echo "    listen 15672;" | sudo tee -a $RABBITMQ_CONFIG_FILE
echo "    location / {" | sudo tee -a $RABBITMQ_CONFIG_FILE
echo "        proxy_pass http://$RABBITMQ_IP:15672;" | sudo tee -a $RABBITMQ_CONFIG_FILE
echo "        include proxy_params;" | sudo tee -a $RABBITMQ_CONFIG_FILE
echo "    }" | sudo tee -a $RABBITMQ_CONFIG_FILE
echo "}" | sudo tee -a $RABBITMQ_CONFIG_FILE

sudo ln -s $RABBITMQ_CONFIG_FILE /etc/nginx/sites-enabled/

sudo nginx -t
if [ "$?" -ne "0" ]; then
    echo nginx config error, exiting
    exit $?
fi

sudo systemctl restart nginx