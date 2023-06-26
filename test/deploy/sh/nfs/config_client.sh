# Make it as idempotent as possible, it can be called over and over

if [ "$NFS_DIRS" = "" ]; then
  echo Error, missing: NFS_DIRS=/mnt/capi_cfg,/mnt/capi_in,/mnt/capi_out
  exit 1
fi
if [ "$INTERNAL_BASTION_IP" = "" ]; then
  echo Error, missing: INTERNAL_BASTION_IP=10.5.0.10
  exit 1
fi
if [ "$SSH_USER" = "" ]; then
  echo Error, missing: SSH_USER=ubuntu
  exit 1
fi

for i in ${NFS_DIRS//,/ }
do
  if [ ! -d $i ]; then
    sudo mkdir $i
  fi
  sudo chown $SSH_USER $i # Requires no_root_squash on the NFS server side
  if ! grep -Fxq "$i" /etc/fstab; then
    echo "$INTERNAL_BASTION_IP:$i $i nfs defaults 0 0" | sudo tee -a /etc/fstab
  fi
  sudo mount $i
  sudo mount $INTERNAL_BASTION_IP:$i
done

# Unmount of needed:
# sudo umount -f /mnt/capi_cfg