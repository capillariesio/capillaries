resource "aws_network_interface" "cassandra_internal_ip" {
  count         = var.number_of_cassandra_hosts
  subnet_id     = aws_subnet.private_subnet.id
  private_ips   = [format("10.5.0.%02s", count.index+11)] // 10.5.0.11, 10.5.0.12, ...
  security_groups = [aws_security_group.capillaries_securitygroup_private.id]

  tags = {
    Name = format("capillaries_cassandra_internal_ip_%02s", count.index+11)
  }
}

locals {
  cassandra_provisioner_common_vars = "SSH_USER=${var.ssh_user} S3_LOG_URL=${var.s3_log_url} JMX_EXPORTER_VERSION=${var.jmx_exporter_version} PROMETHEUS_NODE_EXPORTER_VERSION=${var.prometheus_node_exporter_version} CASSANDRA_VERSION=${var.cassandra_version} CASSANDRA_HOSTS=${local.cassandra_hosts}"
  cassandra_internal_ip_map   = { for i in range(var.number_of_cassandra_hosts): i => format("CASSANDRA_INTERNAL_IP=10.5.0.%02s", i+11) }
  cassandra_initial_token_map = { for i in range(var.number_of_cassandra_hosts): i => format("CASSANDRA_INITIAL_TOKEN=%s", local.cassandra_initial_tokens[i]) }
  cassandra_nvme_regex_map    = { for i in range(var.number_of_cassandra_hosts): i => format("CASSANDRA_NVME_REGEX=\"%s\"", var.nvme_regex_map[var.cassandra_instance_type]) }
  cassandra_provisioner_vars  = { for i in range(var.number_of_cassandra_hosts):
    i => join(" ", [local.cassandra_provisioner_common_vars],
      [local.cassandra_internal_ip_map[tostring(i)]],
      [local.cassandra_initial_token_map[tostring(i)]],
      [local.cassandra_nvme_regex_map[tostring(i)]] ) }
}

resource "aws_instance" "cassandra" {
  instance_type          = var.cassandra_instance_type
  ami                    = var.cassandra_ami_name
  count                  = var.number_of_cassandra_hosts
  key_name               = var.ssh_keypair_name
  network_interface {
    network_interface_id = aws_network_interface.cassandra_internal_ip[count.index].id
    device_index         = 0
  }
  # Cassandra node needs to assume this role to access S3 bucket to get cloud-init cassandra.sh
  iam_instance_profile = aws_iam_instance_profile.capillaries_instance_profile.name

  user_data              = templatefile("./cassandra.sh.tpl", {
    os_arch                                = var.os_arch
    ssh_user                               = var.ssh_user
    cassandra_provisioner_vars             = local.cassandra_provisioner_vars[count.index]
    capillaries_tf_deploy_temp_bucket_name = var.capillaries_tf_deploy_temp_bucket_name
  })  
 
  tags = {
    Name = format("capillaries_cassandra_%02s", count.index+11)
  }
}

output "output_cassandra_provisioner_vars" {
  value = local.cassandra_provisioner_vars
}

