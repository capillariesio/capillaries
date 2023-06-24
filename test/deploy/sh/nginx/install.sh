# https://www.digitalocean.com/community/tutorials/how-to-configure-nginx-as-a-reverse-proxy-on-ubuntu-22-04

sudo apt-get -y install nginx

# Remove nginx stub site
sudo rm -f /etc/nginx/sites-enabled/default

sudo nginx -t
if [ "$?" -ne "0" ]; then
    echo nginx config error, exiting
    exit $?
fi

sudo systemctl restart nginx