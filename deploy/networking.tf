resource "aws_vpc" "main_vpc" {
  cidr_block = "10.5.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support = true
  tags = {
    "Name" = "capillaries_main_vpc"
  }
}

resource "aws_subnet" "private_subnet" {
  availability_zone = var.awsavailabilityzone
  cidr_block = "10.5.0.0/24"
  vpc_id = aws_vpc.main_vpc.id
  tags = {
    "Name" = "capillaries_private_subnet"
  }
}

resource "aws_subnet" "public_subnet" {
  availability_zone = var.awsavailabilityzone
  cidr_block = "10.5.1.0/24"
  vpc_id = aws_vpc.main_vpc.id
  tags = {
    "Name" = "capillaries_public_subnet"
  }
}

resource "aws_security_group" "capillaries_securitygroup_bastion" {
  name = "capillaries_securitygroup_bastion"
  description = "capillaries_securitygroup_bastion"
  vpc_id = aws_vpc.main_vpc.id
  tags = {
    "Name" = "capillaries_securitygroup_bastion"
  }
}

resource "aws_security_group" "capillaries_securitygroup_private" {
  name = "capillaries_securitygroup_private"
  description = "capillaries_securitygroup_private"
  vpc_id = aws_vpc.main_vpc.id
  tags = {
    "Name" = "capillaries_securitygroup_private"
  }
}

# Bastion

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_bastion_ssh" {
  description = "External SSH"
  security_group_id = aws_security_group.capillaries_securitygroup_bastion.id
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = 22
  ip_protocol       = "tcp"
  to_port           = 22
}

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_bastion_webapi_external" {
  description = "Capillaries WebAPI external port"
  security_group_id = aws_security_group.capillaries_securitygroup_bastion.id
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = var.external_webapi_port
  ip_protocol       = "tcp"
  to_port           = var.external_webapi_port
}

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_bastion_capiui" {
  description = "Capillaries UI external port"
  security_group_id = aws_security_group.capillaries_securitygroup_bastion.id
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = 80
  ip_protocol       = "tcp"
  to_port           = 80
}

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_bastion_prometheus_node_exporter" {
  description = "Capillaries Prometheus node exporter"
  security_group_id = aws_security_group.capillaries_securitygroup_bastion.id
  cidr_ipv4         = aws_vpc.main_vpc.cidr_block
  from_port         = 9100
  ip_protocol       = "tcp"
  to_port           = 9100
}

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_bastion_prometheus_go_webapi_exporter" {
  description = "Prometheus Go webapi exporter"
  security_group_id = aws_security_group.capillaries_securitygroup_bastion.id
  cidr_ipv4         = aws_vpc.main_vpc.cidr_block
  from_port         = 9200
  ip_protocol       = "tcp"
  to_port           = 9200
}

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_bastion_capimq_broker_internal" {
  description = "CapiMQ API port (internal)"
  security_group_id = aws_security_group.capillaries_securitygroup_bastion.id
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = var.internal_capimq_broker_port
  ip_protocol       = "tcp"
  to_port           = var.internal_capimq_broker_port
}

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_bastion_capimq_broker_external" {
  description = "CapiMQ API port (external)"
  security_group_id = aws_security_group.capillaries_securitygroup_bastion.id
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = var.external_capimq_broker_port
  ip_protocol       = "tcp"
  to_port           = var.external_capimq_broker_port
}

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_bastion_amqp10_api" {
  description = "ActiveMQ/RabbitMQ API"
  security_group_id = aws_security_group.capillaries_securitygroup_bastion.id
  cidr_ipv4         = aws_vpc.main_vpc.cidr_block
  from_port         = 5672
  ip_protocol       = "tcp"
  to_port           = 5672
}

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_bastion_activemq_console_external" {
  description = "ActiveMQ console (external)"
  security_group_id = aws_security_group.capillaries_securitygroup_bastion.id
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = var.external_activemq_console_port
  ip_protocol       = "tcp"
  to_port           = var.external_activemq_console_port
}

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_bastion_rabbitmqmq_console_external" {
  description = "RabbitMQ console (external)"
  security_group_id = aws_security_group.capillaries_securitygroup_bastion.id
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = var.external_rabbitmq_console_port
  ip_protocol       = "tcp"
  to_port           = var.external_rabbitmq_console_port
}

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_bastion_prometheus_console_external" {
  description = "Prometheus console (external)"
  security_group_id = aws_security_group.capillaries_securitygroup_bastion.id
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = var.external_prometheus_console_port
  ip_protocol       = "tcp"
  to_port           = var.external_prometheus_console_port
}

resource "aws_vpc_security_group_egress_rule" "capillaries_sg_bastion_egress_all" {
  description = "Allow all outbound traffic"
  security_group_id = aws_security_group.capillaries_securitygroup_bastion.id
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = 0
  ip_protocol       = "-1"
  to_port           = 0
}

# Internal

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_private_ssh" {
  description = "Internal SSH"
  security_group_id = aws_security_group.capillaries_securitygroup_private.id
  cidr_ipv4         = aws_vpc.main_vpc.cidr_block
  from_port         = 22
  ip_protocol       = "tcp"
  to_port           = 22
}

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_private_prometheus_node_exporter" {
  description = "Prometheus node exporter"
  security_group_id = aws_security_group.capillaries_securitygroup_private.id
  cidr_ipv4         = aws_vpc.main_vpc.cidr_block
  from_port         = 9100
  ip_protocol       = "tcp"
  to_port           = 9100
}

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_private_prometheus_go_daemon_exporter" {
  description = "Prometheus Go daemon exporter"
  security_group_id = aws_security_group.capillaries_securitygroup_private.id
  cidr_ipv4         = aws_vpc.main_vpc.cidr_block
  from_port         = 9200
  ip_protocol       = "tcp"
  to_port           = 9200
}

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_private_jmx_exporter" {
  description = "JMX exporter"
  security_group_id = aws_security_group.capillaries_securitygroup_private.id
  cidr_ipv4         = aws_vpc.main_vpc.cidr_block
  from_port         = 7070
  ip_protocol       = "tcp"
  to_port           = 7070
}

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_private_cassandra_jmx" {
  description = "Cassandra JMX"
  security_group_id = aws_security_group.capillaries_securitygroup_private.id
  cidr_ipv4         = aws_vpc.main_vpc.cidr_block
  from_port         = 7199
  ip_protocol       = "tcp"
  to_port           = 7199
}

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_private_cassandra_cluster_comm" {
  description = "Cassandra cluster comm"
  security_group_id = aws_security_group.capillaries_securitygroup_private.id
  cidr_ipv4         = aws_vpc.main_vpc.cidr_block
  from_port         = 7000
  ip_protocol       = "tcp"
  to_port           = 7000
}

resource "aws_vpc_security_group_ingress_rule" "capillaries_sg_private_cassandra_api" {
  description = "Cassandra API"
  security_group_id = aws_security_group.capillaries_securitygroup_private.id
  cidr_ipv4         = aws_vpc.main_vpc.cidr_block
  from_port         = 9042
  ip_protocol       = "tcp"
  to_port           = 9042
}

resource "aws_vpc_security_group_egress_rule" "capillaries_sg_private_egress_all" {
  description = "Allow all outbound traffic"
  security_group_id = aws_security_group.capillaries_securitygroup_private.id
  cidr_ipv4         = "0.0.0.0/0"
  from_port         = 0
  ip_protocol       = "-1"
  to_port           = 0
}


resource "aws_eip" "bastion_public_ip" {
  domain = "vpc"
  tags = {
    Name = "capillaries_bastion_public_ip"
  }
}

resource "aws_eip" "natgw_public_ip" {
	domain   = "vpc"
  tags = {
    Name = "capillaries_natgw_public_ip"
  }
}

resource "aws_nat_gateway" "natgw" {
  allocation_id = aws_eip.natgw_public_ip.id
  subnet_id     = aws_subnet.public_subnet.id

  tags = {
    Name = "capillaries_natgw"
  }
  depends_on = [aws_internet_gateway.igw]
}

resource "aws_route_table" "private_subnet_rt_to_natgw" {
 vpc_id = aws_vpc.main_vpc.id
 
 route {
   cidr_block = "0.0.0.0/0"
   gateway_id = aws_nat_gateway.natgw.id
 }
 
 tags = {
   Name = "capillaries_private_subnet_rt_to_natgw"
 }
}

resource "aws_route_table_association" "private_subnet_rt_to_natgw_with_private_subnet" {
 subnet_id      = aws_subnet.private_subnet.id
 route_table_id = aws_route_table.private_subnet_rt_to_natgw.id
}

resource "aws_internet_gateway" "igw" {
  vpc_id = aws_vpc.main_vpc.id

  tags = {
    Name = "capillaries_igw"
  }
}

resource "aws_route_table" "public_subnet_rt_to_igw" {
 vpc_id = aws_vpc.main_vpc.id
 
 route {
   cidr_block = "0.0.0.0/0"
   gateway_id = aws_internet_gateway.igw.id
 }
 
 tags = {
   Name = "capillaries_public_subnet_rt_to_igw"
 }
}

resource "aws_route_table_association" "public_subnet_rt_to_igw_with_public_subnet" {
 subnet_id      = aws_subnet.public_subnet.id
 route_table_id = aws_route_table.public_subnet_rt_to_igw.id
}
