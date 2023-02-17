pkill -2 capidaemon
processid=$(pgrep capidaemon)
if [ "$processid" != "" ]; then
  pkill -9 capidaemon
fi