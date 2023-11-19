# Tweak UI so it calls the proper capiwebapi URL

if [ "$WEBAPI_PORT" = "" ]; then
  echo Error, missing: WEBAPI_PORT=6543
 exit 1
fi
if [ "$SSH_USER" = "" ]; then
  echo Error, missing: SSH_USER=ubuntu
 exit 1
fi

if [ "$EXTERNAL_IP_ADDRESS" = "" ]; then
  echo Error, missing EXTERNAL_IP_ADDRESS=1.2.3.4
  exit 1
fi

echo Patching WebUI to use Webapi ip:port $EXTERNAL_IP_ADDRESS:$WEBAPI_PORT

sed -i -e 's~localhost:6543~'$EXTERNAL_IP_ADDRESS':'$WEBAPI_PORT'~g' /home/$SSH_USER/ui/_app/immutable/nodes/*.js