if [ "$OWNER_USER" = "" ]; then
  echo Error, missing: OWNER_USER=sftpuser or OWNER_USER=ubuntu
  exit 1
fi

rm -f /mnt/capi_in/portfolio_bigtest/*.csv /mnt/capi_in/portfolio_bigtest/*.parquet

sudo tar -zxf /mnt/capi_in/portfolio_bigtest/all.tgz --directory /mnt/capi_in/portfolio_bigtest
rm /mnt/capi_in/portfolio_bigtest/all.tgz

sudo chown $OWNER_USER /mnt/capi_in/portfolio_bigtest/*.csv
sudo chmod 644 /mnt/capi_in/portfolio_bigtest/*.csv
sudo chown $OWNER_USER /mnt/capi_in/portfolio_bigtest/*.parquet
sudo chmod 644 /mnt/capi_in/portfolio_bigtest/*.parquet

