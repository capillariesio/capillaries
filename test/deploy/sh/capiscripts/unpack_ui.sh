if [ "$UI_ROOT" = "" ]; then
  echo Error, missing: UI_ROOT=/home/$SSH_USER/ui
  exit 1
fi

cd $UI_ROOT
tar -xvzf all.tgz
