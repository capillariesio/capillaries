processid=$(pgrep capiwebapi)
if [ "$processid" = "" ]; then
  /home/ubuntu/bin/capiwebapi >> /var/log/capiwebapi/capiwebapi.log 2>&1 &
fi