resource "aws_network_interface" "bastion_internal_ip" {
  subnet_id   = aws_subnet.public_subnet.id
  private_ips = [var.internal_bastion_ip]
  security_groups = [aws_security_group.capillaries_securitygroup_bastion.id]

  tags = {
    Name = "capillaries_bastion_internal_ip"
  }
}

resource "aws_iam_instance_profile" "capillaries_instance_profile" {
  name = "capillaries_instance_profile"
  role = aws_iam_role.capillaries_assume_role.name
}

resource "aws_eip_association" "bastion_public_ip_association" {
  instance_id   = aws_instance.bastion.id
  allocation_id = aws_eip.bastion_public_ip.id
}

resource "aws_instance" "bastion" {
   instance_type          = var.bastion_instance_type
   ami                    = var.bastion_ami_name
   key_name               = var.ssh_keypair_name

   network_interface {
     network_interface_id = aws_network_interface.bastion_internal_ip.id
     device_index         = 0
   }
   iam_instance_profile = aws_iam_instance_profile.capillaries_instance_profile.name
   user_data              = templatefile("./bastion.sh.tpl", {
      awsregion                   = var.awsregion
      ssh_user                    = var.ssh_user
      capillaries_release_url     = var.capillaries_release_url
      os_arch                     = var.os_arch

      bastion_external_ip_address = aws_eip.bastion_public_ip.public_ip
      bastion_allowed_ips         = var.BASTION_ALLOWED_IPS
      internal_webapi_port        = var.internal_webapi_port
      external_webapi_port        = var.external_webapi_port
      external_rabbitmq_console_port = var.external_rabbitmq_console_port
      external_prometheus_console_port = var.external_prometheus_console_port

      cassandra_hosts             = local.cassandra_hosts
      cassandra_port              = var.cassandra_port
      cassandra_username          = var.cassandra_username
      cassandra_password          = var.cassandra_password

      rabbitmq_erlang_version_amd64 = var.rabbitmq_erlang_version_amd64
      rabbitmq_server_version_amd64 = var.rabbitmq_server_version_amd64
      rabbitmq_erlang_version_arm64 = var.rabbitmq_erlang_version_arm64
      rabbitmq_server_version_arm64 = var.rabbitmq_server_version_arm64
      rabbitmq_admin_name           = var.rabbitmq_admin_name
      rabbitmq_admin_pass           = var.rabbitmq_admin_pass
      rabbitmq_user_name            = var.rabbitmq_user_name
      rabbitmq_user_pass            = var.rabbitmq_user_pass

      rabbitmq_url                  = local.rabbitmq_url
      prometheus_node_exporter_version = var.prometheus_node_exporter_version
      prometheus_server_version     = var.prometheus_server_version
      prometheus_targets            = local.prometheus_targets

      s3_log_url                    = var.s3_log_url
   })
  
   tags = {
       Name = "capillaries_bastion"
   }
}

   output "output_bastion_public_ip" {
     value = aws_eip.bastion_public_ip.public_ip
   }