# Expecting
# MOUNT_POINT_CFG=/mnt/capi_cfg
# MOUNT_POINT_IN=/mnt/capi_in
# MOUNT_POINT_OUT=/mnt/capi_out

for filepath in $(find $MOUNT_POINT_CFG -name 'script_params*.json' -type f -print); do
  sudo sed -i -e 's~/tmp/capitest_cfg~'$MOUNT_POINT_CFG'~g' $filepath
  sudo sed -i -e 's~/tmp/capitest_in~'$MOUNT_POINT_IN'~g' $filepath
  sudo sed -i -e 's~/tmp/capitest_out~'$MOUNT_POINT_OUT'~g' $filepath
done
