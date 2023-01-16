useradd -m -d /home/sftpuser -s /bin/bash sftpuser
mkdir /home/sftpuser/.ssh
echo "ssh-rsa ..." > /home/sftpuser/.ssh/authorized_keys
chmod 600 ~/.ssh/sftpuser
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sftpuser sftpuser@ip_of_the_host


# Expecting
MOUNT_POINT_CFG_LOCAL=/home/sftpuser/capi_cfg
MOUNT_POINT_CFG=sftp://sftpuser@158.69.59.166:22/home/sftpuser/capi_cfg
MOUNT_POINT_IN=sftp://sftpuser@158.69.59.166:22/home/sftpuser/capi_in
MOUNT_POINT_OUT=sftp://sftpuser@158.69.59.166:22/home/sftpuser/capi_out

# No sudo here, we have logged in as sftp user and cannot elevate the privileges
for filepath in $(find $MOUNT_POINT_CFG_LOCAL -name 'script_params*.json' -type f -print); do
  sed -i -e 's~/tmp/capitest_cfg~'$MOUNT_POINT_CFG'~g' $filepath
  sed -i -e 's~/tmp/capitest_in~'$MOUNT_POINT_IN'~g' $filepath
  sed -i -e 's~/tmp/capitest_out~'$MOUNT_POINT_OUT'~g' $filepath
done


test_tag_and_denormalize
sftp://sftpuser@158.69.59.166/home/sftpuser/capi_cfg/tag_and_denormalize/script.json
sftp://sftpuser@158.69.59.166/home/sftpuser/capi_cfg/tag_and_denormalize/script_params_one_run.json
read_tags,read_products
tag_totals
