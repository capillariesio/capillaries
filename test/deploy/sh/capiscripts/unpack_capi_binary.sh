if [ "$CAPI_BINARY" = "" ]; then
  echo Error, missing: CAPI_BINARY=/home/$SSH_USER/bin/capitoolbelt
  exit 1
fi

gzip -d -f $CAPI_BINARY.gz
chmod 744 $CAPI_BINARY

