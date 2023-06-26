pkill -2 capidaemon
processid=$(pgrep capidaemon)
if [ "$processid" != "" ]; then
  echo Trying pkill -9...
  pkill -9 capidaemon 2> /dev/null
  processid=$(pgrep capidaemon)
  if [ "$processid" != "" ]; then
    echo pkill -9 did not kill
    exit 9
  fi 
fi