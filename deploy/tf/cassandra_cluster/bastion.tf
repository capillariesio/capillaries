resource "aws_network_interface" "bastion_internal_ip" {
  subnet_id   = aws_subnet.public_subnet.id
  private_ips = [var.internal_bastion_ip]
  security_groups = [aws_security_group.capillaries_securitygroup_bastion.id]

  tags = {
    Name = "capillaries_bastion_internal_ip"
  }
}

resource "aws_eip_association" "bastion_public_ip_association" {
  instance_id   = aws_instance.bastion.id
  allocation_id = aws_eip.bastion_public_ip.id
}

locals {
  bastion_provisioner_static_vars = "AWSREGION=${var.awsregion} SSH_USER=${var.ssh_user} OS_ARCH=${var.os_arch} CAPILLARIES_RELEASE_URL=${var.capillaries_release_url} EXTERNAL_WEBAPI_PORT=${var.external_webapi_port} INTERNAL_WEBAPI_PORT=${var.internal_webapi_port} EXTERNAL_RABBITMQ_CONSOLE_PORT=${var.external_rabbitmq_console_port} EXTERNAL_PROMETHEUS_CONSOLE_PORT=${var.external_prometheus_console_port} BASTION_ALLOWED_IPS=${var.BASTION_ALLOWED_IPS} S3_LOG_URL=${var.s3_log_url} CASSANDRA_HOSTS=${local.cassandra_hosts} CASSANDRA_PORT=${var.cassandra_port} CASSANDRA_USERNAME=${var.cassandra_username} CASSANDRA_PASSWORD=${var.cassandra_password} RABBITMQ_URL=${local.rabbitmq_url} RABBITMQ_USER_NAME=${var.rabbitmq_user_name} RABBITMQ_USER_PASS=${var.rabbitmq_user_pass} RABBITMQ_ADMIN_NAME=${var.rabbitmq_admin_name} RABBITMQ_ADMIN_PASS=${var.rabbitmq_admin_pass} RABBITMQ_ERLANG_VERSION_AMD64=${var.rabbitmq_erlang_version_amd64} RABBITMQ_SERVER_VERSION_AMD64=${var.rabbitmq_server_version_amd64} RABBITMQ_ERLANG_VERSION_ARM64=${var.rabbitmq_erlang_version_arm64} RABBITMQ_SERVER_VERSION_ARM64=${var.rabbitmq_server_version_arm64} PROMETHEUS_NODE_EXPORTER_VERSION=${var.prometheus_node_exporter_version} PROMETHEUS_SERVER_VERSION=${var.prometheus_server_version} PROMETHEUS_NODE_TARGETS=${local.prometheus_node_targets} PROMETHEUS_JMX_TARGETS=${local.prometheus_jmx_targets} PROMETHEUS_GO_TARGETS=${local.prometheus_go_targets}"
}

resource "aws_instance" "bastion" {
  instance_type          = var.bastion_instance_type
  ami                    = var.bastion_ami_name
  key_name               = var.ssh_keypair_name

  network_interface {
    network_interface_id = aws_network_interface.bastion_internal_ip.id
    device_index         = 0
  }
  
  # Bastion needs to assume this role to access S3 bucket to get cloud-init bastion.sh
  iam_instance_profile = aws_iam_instance_profile.capillaries_instance_profile.name

  user_data              = templatefile("./bastion.sh.tpl", {
    os_arch                                = var.os_arch
    ssh_user                               = var.ssh_user
    capillaries_instance_profile           = aws_iam_instance_profile.capillaries_instance_profile.name
    bastion_provisioner_vars               = join(" ", [local.bastion_provisioner_static_vars], ["BASTION_EXTERNAL_IP_ADDRESS=${aws_eip.bastion_public_ip.public_ip}"])
    capillaries_tf_deploy_temp_bucket_name = var.capillaries_tf_deploy_temp_bucket_name
  })
  
  tags = {
    Name = "capillaries_bastion"
  }
}
