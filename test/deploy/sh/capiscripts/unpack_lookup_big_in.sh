# Expects
# SFTP_USER=sftpuser
rm -f /mnt/capi_in/lookup_bigtest/*.csv
sudo tar -zxf /mnt/capi_in/lookup_bigtest/all.tgz --directory /mnt/capi_in/lookup_bigtest
sudo chown $SFTP_USER /mnt/capi_in/lookup_bigtest/*.csv
sudo chmod 644 /mnt/capi_in/lookup_bigtest/*.csv