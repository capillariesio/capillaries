if [ "$SSH_USER" = "" ]; then
  echo Error, missing: SSH_USER=ubuntu
 exit 1
fi

processid=$(pgrep capidaemon)
if [ "$processid" = "" ]; then
  /home/$SSH_USER/bin/capidaemon >> /var/log/capidaemon/capidaemon.log 2>&1 &
fi