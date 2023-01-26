if [ "$CAPI_BINARY" = "" ]; then
  echo Error, missing: CAPI_BINARY=/home/$SSH_USER/bin/capitoolbelt
  exit
fi

gzip -d $CAPI_BINARY.gz
chmod 744 $CAPI_BINARY

