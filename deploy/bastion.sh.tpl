#!/bin/bash

echo Running bastion.sh.tpl in $(pwd)...

NAT Gateway init may take up to 2 min.
wh
while true; do
  if ping -q -c 1 -W 1 8.8.8.8 > /dev/null 2>&1; then
    echo "Internet is available."
    break
  else
    echo "Internet is not available. Waiting..."
    sleep 5
  fi
done

sleep 5

# ubuntu.com may be failing
while true; do
  if sudo DEBIAN_FRONTEND=noninteractive apt-get update -y > /dev/null 2>&1; then
    echo "Updated ubuntu"
    break
  else
    echo "Ubuntu update failed. Waiting..."
    sleep 5
  fi
done

sleep 2

while true; do
  if sudo DEBIAN_FRONTEND=noninteractive apt-get install -y unzip > /dev/null 2>&1; then
    echo "Installed unzip"
    break
  else
    echo "unzip installation failed. Waiting..."
    sleep 5
  fi
done

pushd /tmp
if [ "${os_arch}" = "linux/arm64" ]; then
  curl -Ls https://awscli.amazonaws.com/awscli-exe-linux-aarch64.zip -o awscliv2.zip --retry 5 --retry-delay 2 --retry-all-errors
fi
if [ "${os_arch}" = "linux/amd64" ]; then
  curl -Ls https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip -o awscliv2.zip --retry 5 --retry-delay 2 --retry-all-errors
fi
unzip awscliv2.zip
sudo ./aws/install
rm -fR aws
rm awscliv2.zip
popd

# Ensure instance profile is ready
echo Downloading s3://${capillaries_tf_deploy_temp_bucket_name}/bastion.sh ...
for (( i=0; i<10; i++ )); do
    if sudo aws s3 cp s3://${capillaries_tf_deploy_temp_bucket_name}/bastion.sh /tmp/; then
      echo Instance profile is ready
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

sudo chmod +x /tmp/bastion.sh
sudo chown ${ssh_user} /tmp/bastion.sh
echo Running as ${ssh_user}: '${bastion_provisioner_vars} /tmp/bastion.sh > /tmp/bastion.out 2>/tmp/bastion.err'
sudo su ${ssh_user} -c '${bastion_provisioner_vars} /tmp/bastion.sh > /tmp/bastion.out 2>/tmp/bastion.err'
