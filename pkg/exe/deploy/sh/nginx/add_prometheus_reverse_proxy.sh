# nginx reverse proxy
# https://www.digitalocean.com/community/tutorials/how-to-configure-nginx-as-a-reverse-proxy-on-ubuntu-22-04

# Expecting
# PROMETHEUS_IP=10.5.0.4

PROMETHEUS_CONFIG_FILE=/etc/nginx/sites-available/prometheus
sudo rm -f $PROMETHEUS_CONFIG_FILE
sudo touch $PROMETHEUS_CONFIG_FILE

echo "server {" | sudo tee -a $PROMETHEUS_CONFIG_FILE
echo "    listen 9090;" | sudo tee -a $PROMETHEUS_CONFIG_FILE
echo "    location / {" | sudo tee -a $PROMETHEUS_CONFIG_FILE
echo "        proxy_pass http://$PROMETHEUS_IP:9090;" | sudo tee -a $PROMETHEUS_CONFIG_FILE
echo "        include proxy_params;" | sudo tee -a $PROMETHEUS_CONFIG_FILE
echo "    }" | sudo tee -a $PROMETHEUS_CONFIG_FILE
echo "}" | sudo tee -a $PROMETHEUS_CONFIG_FILE

sudo ln -s $PROMETHEUS_CONFIG_FILE /etc/nginx/sites-enabled/

sudo nginx -t
if [ "$?" -ne "0" ]; then
    echo nginx config error, exiting
    exit $?
fi

sudo systemctl restart nginx