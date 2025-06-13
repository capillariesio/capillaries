resource "aws_network_interface" "cassandra_internal_ip" {
  count         = var.number_of_cassandra_hosts
  subnet_id     = aws_subnet.private_subnet.id
  private_ips   = [format("10.5.0.%02s", count.index+11)] // 10.5.0.11, 10.5.0.12, ...
  security_groups = [aws_security_group.capillaries_securitygroup_daemon.id]

  tags = {
    Name = format("capillaries_cassandra_internal_ip_%02s", count.index+11)
  }
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
   user_data              = templatefile("./cassandra.sh.tpl", {
      awsregion                   = var.awsregion
      ssh_user                    = var.ssh_user
      os_arch                     = var.os_arch

      cassandra_hosts             = local.cassandra_hosts
      cassandra_version           = var.cassandra_version
      jmx_exporter_version        = var.jmx_exporter_version
      cassandra_internal_ip       = format("10.5.0.%02s", count.index+11)
      cassandra_initial_token     = local.cassandra_initial_tokens[count.index]
      cassandra_nvme_regex        = var.nvme_regex_map[var.cassandra_instance_type]

      prometheus_node_exporter_version = var.prometheus_node_exporter_version

      s3_log_url                  = var.s3_log_url
   })
  
   tags = {
     Name = format("capillaries_cassandra_%02s", count.index+11)
  }
}

# output "rendered" {
#   value = "${resource.aws_instance.cassandra}"
# }