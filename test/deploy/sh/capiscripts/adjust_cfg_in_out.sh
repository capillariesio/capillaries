if [ "$MOUNT_POINT_CFG_LOCAL" = "" ]; then
  echo Error, missing: MOUNT_POINT_CFG_LOCAL=/home/sftpuser/capi_cfg
 exit 1
fi
if [ "$MOUNT_POINT_CFG" = "" ]; then
  echo Error, missing: MOUNT_POINT_CFG=sftp://sftpuser@bastion_ip_address/home/sftpuser/capi_cfg
 exit 1
fi
if [ "$MOUNT_POINT_IN" = "" ]; then
  echo Error, missing: MOUNT_POINT_IN=sftp://sftpuser@bastion_ip_address/home/sftpuser/capi_in
 exit 1
fi
if [ "$MOUNT_POINT_OUT" = "" ]; then
  echo Error, missing: MOUNT_POINT_OUT=sftp://sftpuser@bastion_ip_address/home/sftpuser/capi_out
 exit 1
fi

for filepath in $(find $MOUNT_POINT_CFG_LOCAL -name 'script_params*.json' -type f -print); do
  sudo sed -i -e 's~/tmp/capi_cfg~'$MOUNT_POINT_CFG'~g' $filepath
  sudo sed -i -e 's~/tmp/capi_in~'$MOUNT_POINT_IN'~g' $filepath
  sudo sed -i -e 's~/tmp/capi_out~'$MOUNT_POINT_OUT'~g' $filepath
done
