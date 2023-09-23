if [ "$NETWORK_CIDR" = "" ]; then
  echo Error, missing: NETWORK_CIDR=10.5.0.0/16
  exit 1
fi

if [ "$NFS_DIRS" = "" ]; then
  echo Error, missing: NFS_DIRS=/mnt/capi_cfg,/mnt/capi_in,/mnt/capi_out
  exit 1
fi

for i in ${NFS_DIRS//,/ }
do
    echo "$i $NETWORK_CIDR(rw,sync,no_subtree_check,no_root_squash)" | sudo tee -a /etc/exports
done

sudo exportfs -a
sudo systemctl restart nfs-kernel-server