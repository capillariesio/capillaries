if [ "$SFTP_USER" = "" ]; then
  echo Error, missing: SFTP_USER=sftpuser
 exit 1
fi

rm -f /mnt/capi_out/lookup_bigtest/*.csv /mnt/capi_out/lookup_bigtest/*.parquet

sudo tar -zxf /mnt/capi_out/lookup_bigtest/all.tgz --directory /mnt/capi_out/lookup_bigtest
rm /mnt/capi_out/lookup_bigtest/all.tgz

sudo chown $SFTP_USER /mnt/capi_out/lookup_bigtest/*.csv
sudo chmod 644 /mnt/capi_out/lookup_bigtest/*.csv
sudo chown $SFTP_USER /mnt/capi_out/lookup_bigtest/*.parquet
sudo chmod 644 /mnt/capi_out/lookup_bigtest/*.parquet

