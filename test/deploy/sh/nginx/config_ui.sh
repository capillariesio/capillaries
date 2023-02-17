if [ "$SSH_USER" = "" ]; then
  echo Error, missing: SSH_USER=ubuntu
 exit 1
fi

UI_CONFIG_FILE=/etc/nginx/sites-available/ui
sudo rm -f $UI_CONFIG_FILE

sudo tee $UI_CONFIG_FILE <<EOF
server {
  listen 80;
  listen [::]:80;
  root /home/$SSH_USER/ui;
  index index.html;
  location / {
  }
}
EOF

sudo chmod 755 /home
sudo chmod 755 /home/$SSH_USER
sudo chmod 755 /home/$SSH_USER/ui

sudo ln -s $UI_CONFIG_FILE /etc/nginx/sites-enabled/

sudo nginx -t
if [ "$?" -ne "0" ]; then
    echo nginx config error, exiting
    exit $?
fi

sudo systemctl restart nginx