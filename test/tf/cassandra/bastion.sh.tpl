#!/bin/bash

echo Running bastion.sh.tpl in $(pwd)...

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

sudo aws s3 cp s3://${capillaries_tf_deploy_temp_bucket_name}/bastion.sh /tmp/
sudo chmod +x /tmp/bastion.sh
sudo chown ${ssh_user} /tmp/bastion.sh
echo Running bastion.sh with ${bastion_provisioner_vars} ...
sudo su ${ssh_user} -c '${bastion_provisioner_vars} /tmp/bastion.sh > /tmp/bastion.out 2>/tmp/bastion.err'
