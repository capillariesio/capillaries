#!/bin/bash

echo Running cassandra.sh.tpl in $(pwd)...

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

echo Downloading s3://${capillaries_tf_deploy_temp_bucket_name}/cassandra.sh ...
sudo aws s3 cp s3://${capillaries_tf_deploy_temp_bucket_name}/cassandra.sh /tmp/
sudo chmod +x /tmp/cassandra.sh
sudo chown ${ssh_user} /tmp/cassandra.sh
echo Running cassandra.sh with ${cassandra_provisioner_vars} as ${ssh_user} ...
sudo su ${ssh_user} -c '${cassandra_provisioner_vars} /tmp/cassandra.sh > /tmp/cassandra.out 2>/tmp/cassandra.err'
