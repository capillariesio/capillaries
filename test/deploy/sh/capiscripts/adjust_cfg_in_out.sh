# Expecting
# MOUNT_POINT_CFG_LOCAL=/home/sftpuser/capi_cfg
# MOUNT_POINT_CFG=sftp://sftpuser@bastion_ip_address/home/sftpuser/capi_cfg
# MOUNT_POINT_IN=sftp://sftpuser@bastion_ip_address/home/sftpuser/capi_in
# MOUNT_POINT_OUT=sftp://sftpuser@bastion_ip_address/home/sftpuser/capi_out

for filepath in $(find $MOUNT_POINT_CFG_LOCAL -name 'script_params*.json' -type f -print); do
  sudo sed -i -e 's~/tmp/capitest_cfg~'$MOUNT_POINT_CFG'~g' $filepath
  sudo sed -i -e 's~/tmp/capitest_in~'$MOUNT_POINT_IN'~g' $filepath
  sudo sed -i -e 's~/tmp/capitest_out~'$MOUNT_POINT_OUT'~g' $filepath
done
