# Tweak UI so it call the proper capiwebapi URL

# Expecting
# WEBAPI_IP=208.113.134.216
# $WEBAPI_PORT=6543

sed -i -e 's~localhost:6543~'$WEBAPI_IP':'$WEBAPI_PORT'~g' /home/ubuntu/ui/build/bundle.js