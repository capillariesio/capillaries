resource "aws_network_interface" "daemon_internal_ip" {
  count         = var.number_of_daemons
  subnet_id     = aws_subnet.private_subnet.id
  private_ips   = [format("10.5.0.1%02s", count.index+1)] // 10.5.0.101, 10.5.0.102, ...
  security_groups = [aws_security_group.capillaries_securitygroup_daemon.id]

  tags = {
    Name = format("capillaries_daemon_internal_ip_1%02s", count.index+1)
  }
}

resource "aws_instance" "daemon" {
   instance_type          = var.bastion_instance_type
   ami                    = var.bastion_ami_name
   count                  = var.number_of_daemons
   key_name               = var.ssh_keypair_name
   network_interface {
     network_interface_id = aws_network_interface.daemon_internal_ip[count.index].id
     device_index         = 0
   }
   iam_instance_profile = aws_iam_instance_profile.capillaries_instance_profile.name
   user_data              = templatefile("./daemon.sh.tpl", {
      awsregion                   = var.awsregion
      ssh_user                    = var.ssh_user
      capillaries_release_url     = var.capillaries_release_url
      os_arch                     = var.os_arch
      cassandra_hosts             = var.cassandra_hosts
      cassandra_port              = var.cassandra_port
      cassandra_username          = var.CASSANDRA_USERNAME
      cassandra_password          = var.CASSANDRA_PASSWORD
	  rabbitmq_url				  = var.RABBITMQ_URL
      s3_log_url                  = var.s3_log_url
   })
  
   tags = {
     Name = format("capillaries_daemon_1%02s", count.index+1)
  }
}