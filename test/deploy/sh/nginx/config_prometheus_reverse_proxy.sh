# nginx reverse proxy
# https://www.digitalocean.com/community/tutorials/how-to-configure-nginx-as-a-reverse-proxy-on-ubuntu-22-04

if [ "$PROMETHEUS_IP" = "" ]; then
  echo Error, missing: PROMETHEUS_IP=10.5.0.4
 exit 1
fi

PROMETHEUS_CONFIG_FILE=/etc/nginx/sites-available/prometheus
sudo rm -f $PROMETHEUS_CONFIG_FILE

sudo tee $PROMETHEUS_CONFIG_FILE <<EOF
server {
    listen 9090;
    location / {
        proxy_pass http://$PROMETHEUS_IP:9090;
        include proxy_params;
    }
}
EOF

sudo ln -s $PROMETHEUS_CONFIG_FILE /etc/nginx/sites-enabled/

sudo nginx -t
if [ "$?" -ne "0" ]; then
    echo nginx config error, exiting
    exit $?
fi

sudo systemctl restart nginx