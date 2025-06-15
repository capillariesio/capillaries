#!/bin/bash

echo Running daemon.sh.tpl in $(pwd)...

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

sudo aws s3 cp s3://${capillaries_tf_deploy_temp_bucket_name}/daemon.sh /tmp/
sudo chmod +x /tmp/daemon.sh
sudo chown ${ssh_user} /tmp/daemon.sh
echo Running daemon.sh with ${daemon_provisioner_vars} ...
sudo su ${ssh_user} -c '${daemon_provisioner_vars} /tmp/daemon.sh > /tmp/daemon.out 2>/tmp/daemon.err'