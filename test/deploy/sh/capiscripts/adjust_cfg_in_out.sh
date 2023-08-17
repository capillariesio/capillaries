# LOCAL_CFG_LOCATION == MOUNT_POINT_CFG for the NFS case
# LOCAL_CFG_LOCATION != MOUNT_POINT_CFG for the SFTP case

if [ "$LOCAL_CFG_LOCATION" = "" ]; then
  echo Error, missing:
  echo "LOCAL_CFG_LOCATION=/home/sftpuser/capi_cfg (data on ~)"
  echo "LOCAL_CFG_LOCATION=/mnt/capi_cfg (data on mounted volume)"
  exit 1
fi
if [ "$MOUNT_POINT_CFG" = "" ]; then
  echo Error, missing:
  echo "MOUNT_POINT_CFG=sftp://sftpuser@bastion_ip_address/home/sftpuser/capi_cfg (sftp at ~) or"
  echo "MOUNT_POINT_CFG=sftp://sftpuser@bastion_ip_address/mnt/capi_cfg (sftp at mounted volume) or"
  echo "MOUNT_POINT_CFG=/mnt/capi_cfg (nfs)"
  exit 1
fi
if [ "$MOUNT_POINT_IN" = "" ]; then
  echo Error, missing:
  echo "MOUNT_POINT_IN=sftp://sftpuser@bastion_ip_address/home/sftpuser/capi_in (sftp at ~)"
  echo "MOUNT_POINT_IN=sftp://sftpuser@bastion_ip_address/mnt/capi_in (sftp at mounted volume) or"
  echo "MOUNT_POINT_IN=/mnt/capi_in (nfs)"
  exit 1
fi
if [ "$MOUNT_POINT_OUT" = "" ]; then
  echo Error, missing:
  echo "MOUNT_POINT_OUT=sftp://sftpuser@bastion_ip_address/home/sftpuser/capi_out (sftp at ~)"
  echo "MOUNT_POINT_OUT=sftp://sftpuser@bastion_ip_address/mnt/capi_out (sftp at mounted volume) or"
  echo "MOUNT_POINT_OUT=/mnt/capi_out (nfs)"
  exit 1
fi

# In all script params files, replace /tmp/... with /mnt/...
for filepath in $(find $LOCAL_CFG_LOCATION -name 'script_params*.json' -type f -print); do
  sudo sed -i -e 's~/tmp/capi_cfg~'$MOUNT_POINT_CFG'~g' $filepath
  sudo sed -i -e 's~/tmp/capi_in~'$MOUNT_POINT_IN'~g' $filepath
  sudo sed -i -e 's~/tmp/capi_out~'$MOUNT_POINT_OUT'~g' $filepath
done
