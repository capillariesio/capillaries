pkill -2 webapi
processid=$(pgrep webapi)
if [ "$processid" -ne "" ]; then
  pkill -9 webapi
fi