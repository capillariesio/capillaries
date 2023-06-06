if [ "$SFTP_USER" = "" ]; then
  echo Error, missing: SFTP_USER=sftpuser
 exit 1
fi

rm -f /mnt/capi_in/lookup_bigtest/*.csv /mnt/capi_in/lookup_bigtest/*.parquet

sudo tar -zxf /mnt/capi_in/lookup_bigtest/all.tgz --directory /mnt/capi_in/lookup_bigtest

sudo chown $SFTP_USER /mnt/capi_in/lookup_bigtest/*.csv
sudo chmod 644 /mnt/capi_in/lookup_bigtest/*.csv
sudo chown $SFTP_USER /mnt/capi_in/lookup_bigtest/*.parquet
sudo chmod 644 /mnt/capi_in/lookup_bigtest/*.parquet

