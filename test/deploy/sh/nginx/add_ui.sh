UI_CONFIG_FILE=/etc/nginx/sites-available/ui
sudo rm -f $UI_CONFIG_FILE

sudo tee $UI_CONFIG_FILE <<EOF
server {
  listen 80;
  listen [::]:80;
  root /home/ubuntu/ui;
  index index.html;
  location / {
  }
}
EOF

sudo chmod 755 /home
sudo chmod 755 /home/ubuntu
sudo chmod 755 /home/ubuntu/ui

sudo ln -s $UI_CONFIG_FILE /etc/nginx/sites-enabled/

# Remove nginx stub site
sudo rm -f /etc/nginx/sites-enabled/default

sudo nginx -t
if [ "$?" -ne "0" ]; then
    echo nginx config error, exiting
    exit $?
fi

sudo systemctl restart nginx