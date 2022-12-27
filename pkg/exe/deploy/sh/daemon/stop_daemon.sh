pkill -2 capidaemon
processid=$(pgrep capidaemon)
if [ "$processid" -ne "" ]; then
  pkill -9 capidaemon
fi