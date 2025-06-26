resource "aws_network_interface" "daemon_internal_ip" {
  count         = var.number_of_daemons
  subnet_id     = aws_subnet.private_subnet.id
  private_ips   = [format("10.5.0.1%02s", count.index+1)] // 10.5.0.101, 10.5.0.102, ...
  security_groups = [aws_security_group.capillaries_securitygroup_private.id]

  tags = {
    Name = format("capillaries_daemon_internal_ip_1%02s", count.index+1)
  }
}

locals {
  daemon_provisioner_vars = "DAEMON_GOMEMLIMIT_GB=${local.daemon_gomemlimit_gb} DAEMON_GOGC=${var.daemon_gogc} AWSREGION=${var.awsregion} SSH_USER=${var.ssh_user} OS_ARCH=${var.os_arch} CAPILLARIES_RELEASE_URL=${var.capillaries_release_url} S3_LOG_URL=${var.s3_log_url} CASSANDRA_HOSTS=${local.cassandra_hosts} CASSANDRA_PORT=${var.cassandra_port} CASSANDRA_USERNAME=${var.cassandra_username} CASSANDRA_PASSWORD=${var.cassandra_password} RABBITMQ_URL=${local.rabbitmq_url} RABBITMQ_USER_NAME=${var.rabbitmq_user_name} RABBITMQ_USER_PASS=${var.rabbitmq_user_pass} PROMETHEUS_NODE_EXPORTER_VERSION=${var.prometheus_node_exporter_version} WRITER_WORKERS=${local.daemon_writer_workers} THREAD_POOL_SIZE=${local.daemon_thread_pool_size}"
}

resource "aws_instance" "daemon" {
  instance_type          = var.daemon_instance_type
  ami                    = var.daemon_ami_name
  count                  = var.number_of_daemons
  key_name               = var.ssh_keypair_name
  network_interface {
    network_interface_id = aws_network_interface.daemon_internal_ip[count.index].id
    device_index         = 0
  }

  # Daemon needs to assume this role to access S3 bucket to get cloud-init daemon.sh and read/write data
  iam_instance_profile = aws_iam_instance_profile.capillaries_instance_profile.name
  
  user_data              = templatefile("./daemon.sh.tpl", {
    os_arch                                = var.os_arch
    ssh_user                               = var.ssh_user
    capillaries_instance_profile           = aws_iam_instance_profile.capillaries_instance_profile.name
    daemon_provisioner_vars                = local.daemon_provisioner_vars
    capillaries_tf_deploy_temp_bucket_name = var.capillaries_tf_deploy_temp_bucket_name
  })  
   tags = {
     Name = format("capillaries_daemon_1%02s", count.index+1)
  }
}
