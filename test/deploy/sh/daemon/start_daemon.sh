processid=$(pgrep capidaemon)
if [ "$processid" = "" ]; then
  /home/ubuntu/bin/capidaemon >> /var/log/capidaemon/capidaemon.log 2>&1 &
fi