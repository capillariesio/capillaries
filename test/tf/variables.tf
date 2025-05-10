variable "awsregion" {
	type    = string
	default = "us-east-1"
}

variable "awsavailabilityzone" {
	type    = string
	default = "us-east-1a"
}

variable "internal_bastion_ip" {
    type        = string
    description = "Bastion IP in the VPC"
    default     = "10.5.1.10"
}

variable "capillaries_release_url" {
	type        = string
	description = "Download Capillaries binaries here"
    default     =  "https://capillaries-release.s3.us-east-1.amazonaws.com/latest"
}

variable "os_arch" {
	type        = string
	description = "linux/arm64 or linux/amd64, matches the AMI"
    default     = "linux/arm64"
}

variable "ssh_user" {
	type    = string
	default = "ubuntu"
}

variable "ssh_keypair_name" {
	type        = string
	description = "Name of the AWS keypair to use for SSH access to instances"
    default     =  "sampledeployment005-root-key"
}

variable "bastion_instance_type" {
	type        = string
    default     = "c7g.large"
}

variable "bastion_ami_name" {
	type        = string
	description = "Expires 2026-12-18 ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-arm64-server-20241218"
    default     =  "ami-04474687c34a061cf"
}

variable "number_of_daemons" {
	type        = number
	default     = 4
}

variable "daemon_instance_type" {
	type        = string
    default     = "c7g.medium"
}

variable "daemon_ami_name" {
	type        = string
	description = "Expires 2026-12-18 ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-arm64-server-20241218"
    default     =  "ami-04474687c34a061cf"
}

variable "daemon_writer_workers" {
	type        = number
    default     = 5
}
variable "daemon_thread_pool_size" {
	type        = number
	description = "Number of Daemon instance CPUs x1.5"
    default     = 3
}

variable "cassandra_hosts" {
	type        = string
	description = "List of cassandra host urls, or one url, no username/password, no port"
	default     = "cassandra.us-east-1.amazonaws.com"
}

variable "cassandra_port" {
	type        = string
	description = "Amazon Keyspaces Cassandra port"
	default     = "9142"
}

variable "internal_webapi_port" {
	type    = string
	default = "6543"
}

variable "external_webapi_port" {
	type    = string
	default = "6544"
}

variable "s3_log_url" {
	type    = string
	default = "s3://capillaries-testbucket/log"
}

# Env variables TF_VAR_

variable "BASTION_ALLOWED_IPS" {
	type        = string
	description = "Comma-separated list of IP addresses and cidr blocks allowed to access bastion from the outside"
}

variable "RABBITMQ_URL" {
	type        = string
	description = "Full url (with username, password, port) of the RabbitMQ broker"
}

variable "CASSANDRA_USERNAME" {
	type        = string
	description = "Cassandra username"
}

variable "CASSANDRA_PASSWORD" {
	type        = string
	description = "Cassandra password"
}
