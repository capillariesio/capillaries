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
      cassandra_hosts             = var.cassandra_hosts
      cassandra_port              = var.cassandra_port
      cassandra_username          = var.CASSANDRA_USERNAME
      cassandra_password          = var.CASSANDRA_PASSWORD
 	    rabbitmq_url                = var.RABBITMQ_URL

      bastion_external_ip_address = aws_eip.bastion_public_ip.public_ip
      bastion_allowed_ips         = var.BASTION_ALLOWED_IPS
      internal_webapi_port        = var.internal_webapi_port
      external_webapi_port        = var.external_webapi_port
      s3_log_url                  = var.s3_log_url
   })
  
   tags = {
       Name = "capillaries_bastion"
   }
}

   output "output_bastion_public_ip" {
     value = aws_eip.bastion_public_ip.public_ip
   }