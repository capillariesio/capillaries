#!/bin/bash

echo Running daemon.sh.tpl in $(pwd)...

# Sometimes NAT gateway is not ready yet, wait until it is
while true; do
  if ping -q -c 1 -W 1 8.8.8.8 > /dev/null 2>&1; then
    echo "Internet is available."
    break
  else
    echo "Internet is not available. Waiting..."
    sleep 5
  fi
done

sudo DEBIAN_FRONTEND=noninteractive apt-get update -y
sudo apt-get install -y unzip
pushd /tmp
if [ "${os_arch}" = "linux/arm64" ]; then
  curl https://awscli.amazonaws.com/awscli-exe-linux-aarch64.zip -o awscliv2.zip
fi
if [ "${os_arch}" = "linux/amd64" ]; then
  curl https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip -o awscliv2.zip
fi
unzip awscliv2.zip
sudo ./aws/install
rm -fR aws
rm awscliv2.zip
popd

# Ensure instance profile is ready
echo Downloading s3://${capillaries_tf_deploy_temp_bucket_name}/daemon.sh ...
for (( i=0; i<10; i++ )); do
    if sudo aws s3 cp s3://${capillaries_tf_deploy_temp_bucket_name}/daemon.sh /tmp/; then
      echo Instance profile is ready, script downloaded
      break
    else
      echo Instance profile NOT ready, attempt $i
      if [ "$i" = "9" ]; then
        echo Giving up trying to check instance profile
        exit 1
      fi
      sleep 5
    fi
done

sudo chmod +x /tmp/daemon.sh
sudo chown ${ssh_user} /tmp/daemon.sh
echo Running as ${ssh_user}: '${daemon_provisioner_vars} /tmp/daemon.sh > /tmp/daemon.out 2>/tmp/daemon.err'
sudo su ${ssh_user} -c '${daemon_provisioner_vars} /tmp/daemon.sh > /tmp/daemon.out 2>/tmp/daemon.err'