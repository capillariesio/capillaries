pkill -2 webapi
processid=$(pgrep webapi)
if [ "$processid" != "" ]; then
  pkill -9 webapi
fi