# nginx reverse proxy
# https://www.digitalocean.com/community/tutorials/how-to-configure-nginx-as-a-reverse-proxy-on-ubuntu-22-04

if [ "$RABBITMQ_IP" = "" ]; then
  echo Error, missing: RABBITMQ_IP=10.5.0.5
 exit 1
fi

RABBITMQ_CONFIG_FILE=/etc/nginx/sites-available/rabbitmq
sudo rm -f $RABBITMQ_CONFIG_FILE

sudo tee $RABBITMQ_CONFIG_FILE <<EOF
server {
    listen 15672;
    location / {
        proxy_pass http://$RABBITMQ_IP:15672;
        include proxy_params;
    }
}
EOF

sudo ln -s $RABBITMQ_CONFIG_FILE /etc/nginx/sites-enabled/

sudo nginx -t
if [ "$?" -ne "0" ]; then
    echo nginx config error, exiting
    exit $?
fi

sudo systemctl restart nginx