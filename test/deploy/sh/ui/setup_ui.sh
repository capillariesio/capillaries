# Tweak UI so it call the proper capiwebapi URL

if [ "$WEBAPI_IP" = "" ]; then
  echo Error, missing: WEBAPI_IP=208.113.134.216
 exit 1
fi
if [ "$WEBAPI_PORT" = "" ]; then
  echo Error, missing: WEBAPI_PORT=6543
 exit 1
fi
if [ "$SSH_USER" = "" ]; then
  echo Error, missing: SSH_USER=ubuntu
 exit 1
fi

sed -i -e 's~localhost:6543~'$WEBAPI_IP':'$WEBAPI_PORT'~g' /home/$SSH_USER/ui/build/bundle.js