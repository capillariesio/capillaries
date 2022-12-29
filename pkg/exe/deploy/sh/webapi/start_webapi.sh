processid=$(pgrep webapi)
if [ "$processid" = "" ]; then
  /home/ubuntu/bin/webapi >> /var/log/webapi/webapi.log 2>&1 &
fi