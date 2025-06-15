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
    default     = "https://capillaries-release.s3.us-east-1.amazonaws.com/latest"
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
    default     = "sampledeployment005-root-key"
}

# variable "ssh_keypair_root_key_file" {
# 	type        = string
# 	description = "Path to local copy of the private key used to access instances via SSH"
#     default     = "~/.ssh/sampledeployment005_rsa"
# }

variable "capillaries_tf_deploy_temp_bucket_name" {
	type = string
	default = "capillaries-tf-deploy-temp-bucket"
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
	default     = 1
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
	description = "From 5 to 20"
    default     = 8
}

variable "number_of_cassandra_hosts" {
	type        = number
	description = "90 max, because IP address starts with 11, and 101 is a daemon"
	default     = 1
}

variable "cassandra_port" {
	type        = string
	description = "Default Cassandra port"
	default     = "9042"
}

variable "cassandra_username" {
	type        = string
	description = "Default Cassandra username"
	default     = "cassandra"
}

variable "cassandra_password" {
	type        = string
	description = "Default Cassandra password"
	default     = "cassandra"
}

variable "cassandra_instance_type" {
	type        = string
	description = "Make sure it's in the nvme_regex_map list"
    default     = "c7gd.large"
}

variable "cassandra_ami_name" {
	type        = string
	description = "Expires 2026-12-18 ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-arm64-server-20241218"
    default     =  "ami-04474687c34a061cf"
}

variable "cassandra_version" {
	type        = string
	description = "4 or 5"
	default     = "50x"
}

# TODO: replace it with 1.3.0
variable "jmx_exporter_version" {
	type        = string
	description = "1.0.1"
	default     = "1.0.1"
}

variable "cassandra_initial_tokens_4" {
    type = list(string)
    default = ["-9223372036854775808", "-4611686018427387904", "0", "4611686018427387904"]
}
# variable "cassandra_initial_tokens_8" {
#     type = list(string)
#     default = ["-9223372036854775808", "-6917529027641081856", "-4611686018427387904", "-2305843009213693952", "0", "2305843009213693952", "4611686018427387904", "6917529027641081856"]
# }
# variable "cassandra_initial_tokens_16" {
#     type = list(string)
#     default = ["-9223372036854775808","-8070450532247928832","-6917529027641081856","-5764607523034234880","-4611686018427387904","-3458764513820540928","-2305843009213693952","-1152921504606846976","0","1152921504606846976","2305843009213693952","3458764513820540928","4611686018427387904","5764607523034234880","6917529027641081856","8070450532247928832"]
# }
# variable "cassandra_initial_tokens_32" {
#     type = list(string)
#     default = ["-9223372036854775808","-8646911284551352320","-8070450532247928832","-7493989779944505344","-6917529027641081856","-6341068275337658368","-5764607523034234880","-5188146770730811392","-4611686018427387904","-4035225266123964416","-3458764513820540928","-2882303761517117440","-2305843009213693952","-1729382256910270464","-1152921504606846976","-576460752303423488","0","576460752303423488","1152921504606846976","1729382256910270464","2305843009213693952","2882303761517117440","3458764513820540928","4035225266123964416","4611686018427387904","5188146770730811392","5764607523034234880","6341068275337658368","6917529027641081856","7493989779944505344","8070450532247928832","8646911284551352320"]
# }

variable "cassandra_initial_tokens_map" {
	type = map(list(string))
	default = {
		"1" = [""]
		"2" = ["-9223372036854775808", "0"]
		"3" = ["-9223372036854775808", "-3074457345618258603", "3074457345618258602"]
		"4" = ["-9223372036854775808", "-4611686018427387904", "0", "4611686018427387904"]
		"8" = ["-9223372036854775808", "-6917529027641081856", "-4611686018427387904", "-2305843009213693952", "0", "2305843009213693952", "4611686018427387904", "6917529027641081856"]
		"16" = ["-9223372036854775808","-8070450532247928832","-6917529027641081856","-5764607523034234880","-4611686018427387904","-3458764513820540928","-2305843009213693952","-1152921504606846976","0","1152921504606846976","2305843009213693952","3458764513820540928","4611686018427387904","5764607523034234880","6917529027641081856","8070450532247928832"]
		"32" = ["-9223372036854775808","-8646911284551352320","-8070450532247928832","-7493989779944505344","-6917529027641081856","-6341068275337658368","-5764607523034234880","-5188146770730811392","-4611686018427387904","-4035225266123964416","-3458764513820540928","-2882303761517117440","-2305843009213693952","-1729382256910270464","-1152921504606846976","-576460752303423488","0","576460752303423488","1152921504606846976","1729382256910270464","2305843009213693952","2882303761517117440","3458764513820540928","4035225266123964416","4611686018427387904","5188146770730811392","5764607523034234880","6341068275337658368","6917529027641081856","7493989779944505344","8070450532247928832","8646911284551352320"]
	}
}

variable "nvme_regex_map" {
  type = map(string)
  default = {
	"c5ad.large"    = "nvme[0-9]n[0-9] [0-9]+.[0-9]G"
    "c5ad.xlarge"   = "nvme[0-9]n[0-9] 139.7G"
    "c5ad.2xlarge"  = "nvme[0-9]n[0-9] 279.4G" # quick_lookup 23s, bastion lsblk: "xvdf 202:80 0 10G  0 disk /mnt/capi_log", cass lsblk: "nvme1n1 259:1 0 139.7G 0 disk"
    "c5ad.4xlarge"  = "nvme[0-9]n[0-9] [0-9]+.[0-9]G" # quick_lookup 23s, cass lsblk: "nvme1n1 259:0 0 279.4G  0 disk /data0"
    "c5ad.8xlarge"  = "nvme[0-9]n[0-9] 558.8G"
    "c5ad.16xlarge" = "nvme[0-9]n[0-9] [0-9]+.[0-9]T"
	"c7gd.large"    = "nvme[0-9]n[0-9] [0-9]+.[0-9]G"
    "c7gd.xlarge"   = "nvme[0-9]n[0-9] 220.7G" # quick_lookup 23s, lsblk: cassandra data0 nvme1n1 220.7G, bastion /mnt/capi_log nvme1n1 10G
    "c7gd.2xlarge"  = "nvme[0-9]n[0-9] [0-9]+.[0-9]G"
    "c7gd.4xlarge"  = "nvme[0-9]n[0-9] 884.8G"
    "c7gd.8xlarge"  = "nvme[0-9]n[0-9] 1.7T"
    "c7gd.16xlarge" = "nvme[0-9]n[0-9] 1.7T"
  }
}

# Number of vCPUs x 1.5
variable "thread_pool_size_map" {
  type = map(number)
  default = {
	"c6a.large"    = 3
    "c6a.xlarge"   = 6
    "c6a.2xlarge"  = 12
    "c6a.4xlarge"  = 24
	"c6a.8xlarge"  = 48
	"c7g.medium"   = 1
	"c7g.large"    = 3
    "c7g.xlarge"   = 6
    "c7g.2xlarge"  = 12
    "c7g.4xlarge"  = 24
    "c7g.8xlarge"  = 48
  }
}

variable "internal_webapi_port" {
	type    = string
	default = "6543"
}

variable "external_webapi_port" {
	type    = string
	default = "6544"
}

variable "external_rabbitmq_console_port" {
	type    = string
	default = "15673"
}

variable "external_prometheus_console_port" {
	type    = string
	default = "9091"
}

variable "s3_log_url" {
	type    = string
	default = "s3://capillaries-testbucket/log"
}

variable "rabbitmq_erlang_version_amd64" {
	type        = string
	description = "Latest Erlang from RabbitMQ team"
	default     = "1:27.2-1"
}

variable "rabbitmq_server_version_amd64" {
	type        = string
	description = "Latest RabbitMQ server from RabbitMQ team"
	default     = "4.0.5-1"
}

variable "rabbitmq_erlang_version_arm64" {
	type        = string
	description = "Ideally, Erlang version should match amd64 releases, but RabbitMQ team is late with arm64 for some reason. Watch RabbitMQ team changing this sometimes as of 2024-2025: 1ubuntu4, 1ubuntu4.1, 1ubuntu4.2."
	default     = "1:25.3.2.8+dfsg-1ubuntu4"
}

variable "rabbitmq_server_version_arm64" {
	type        = string
	description = "Older RabbitMQ server, because newer versions require newer Erlang"
	default     = "3.12.1-1ubuntu1"
}


variable "rabbitmq_admin_name"{
	type        = string
	default     = "radmin"
}

variable "rabbitmq_admin_pass"{
	type        = string
	default     = "rpass"
}

variable "rabbitmq_user_name"{
	type        = string
	default     = "capiuser"
}

variable "rabbitmq_user_pass"{
	type        = string
	default     = "capipass"
}


variable "prometheus_node_exporter_version" {
	type        = string
	default     = "1.9.1"
}

variable "prometheus_server_version" {
	type        = string
	default     = "3.2.1"
}

locals {
	cassandra_hosts          = join(",",[ for i in range(var.number_of_cassandra_hosts) : format("10.5.0.%02s", i+11) ])
    cassandra_initial_tokens = var.cassandra_initial_tokens_map[var.number_of_cassandra_hosts]
	rabbitmq_url             = join("",["amqp://", var.rabbitmq_user_name, ":", var.rabbitmq_user_pass, "@10.5.1.10/"])
    prometheus_targets       = join(",",concat( # "\'localhost:9100\',\'10.5.1.10:9100\'"
		                ["'localhost:9100'"], // bastion node exporter
						[ for i in range(var.number_of_cassandra_hosts) : format("'10.5.0.%02s:9100'", i+11) ], // cassandra node exporters
						[ for i in range(var.number_of_daemons) : format("'10.5.0.1%02s:9100'", i+1) ], // daemon node expoters
						[ for i in range(var.number_of_cassandra_hosts) : format("'10.5.0.%02s:7070'", i+11) ])) // cassandra JMX exporters
} 


# Env variables TF_VAR_

variable "BASTION_ALLOWED_IPS" {
	type        = string
	description = "Comma-separated list of IP addresses and cidr blocks allowed to access bastion from the outside"
}

