pkill -2 capiwebapi
processid=$(pgrep capiwebapi)
if [ "$processid" != "" ]; then
  pkill -9 capiwebapi
fi