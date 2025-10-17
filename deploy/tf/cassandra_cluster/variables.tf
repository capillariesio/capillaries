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
	description = "arm64: ami-04474687c34a061cf Expires 2026-12-18 ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-arm64-server-20241218; amd64: ami-079cb33ef719a7b78 Expires 2026-12-18 ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-20241218 // ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-20240606"
    default     =  "ami-04474687c34a061cf"
}

variable "number_of_daemons" {
	type        = number
	default     = 4
}

variable "daemon_instance_type" {
	type        = string
    default     = "c7g.large"
}

variable "daemon_ami_name" {
	type        = string
	description = "arm64: ami-04474687c34a061cf Expires 2026-12-18 ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-arm64-server-20241218; amd64: ami-079cb33ef719a7b78 Expires 2026-12-18 ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-20241218 // ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-20240606"
    default     =  "ami-04474687c34a061cf"
}

variable "daemon_gogc" {
	type        = string
	description = "GOGC env var for daemon, usually 100"
    default     = "100"
}

variable "number_of_cassandra_hosts" {
	type        = number
	description = "90 max, because IP address starts with 11, and 101 is a daemon"
	default     = 4
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
    default     = "c7gd.2xlarge"
}

variable "cassandra_ami_name" {
	type        = string
	description = "arm64: ami-04474687c34a061cf Expires 2026-12-18 ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-arm64-server-20241218; amd64: ami-079cb33ef719a7b78 Expires 2026-12-18 ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-20241218 // ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-20240606"
    default     = "ami-04474687c34a061cf"
}

variable "cassandra_version" {
	type        = string
	description = "4 or 5"
	default     = "50x"
}

variable "webapi_gogc" {
	type        = string
	description = "GOGC env var for webapi, usually 100"
    default     = "100"
}

variable "jmx_exporter_version" {
	type        = string
	description = "1.5.0"
	default     = "1.5.0"
}

variable "cassandra_initial_tokens_map" {
	type = map(list(string))
	default = {
		"1"  = ["-9223372036854775808"]
		"2"  = ["-9223372036854775808", "0"]
		"3"  = ["-9223372036854775808", "-3074457345618258603", "3074457345618258602"]
		"4"  = ["-9223372036854775808", "-4611686018427387904", "0", "4611686018427387904"]
		"8"  = ["-9223372036854775808", "-6917529027641081856", "-4611686018427387904", "-2305843009213693952", "0", "2305843009213693952", "4611686018427387904", "6917529027641081856"]
		"16" = ["-9223372036854775808","-8070450532247928832","-6917529027641081856","-5764607523034234880","-4611686018427387904","-3458764513820540928","-2305843009213693952","-1152921504606846976","0","1152921504606846976","2305843009213693952","3458764513820540928","4611686018427387904","5764607523034234880","6917529027641081856","8070450532247928832"]
		"32" = ["-9223372036854775808","-8646911284551352320","-8070450532247928832","-7493989779944505344","-6917529027641081856","-6341068275337658368","-5764607523034234880","-5188146770730811392","-4611686018427387904","-4035225266123964416","-3458764513820540928","-2882303761517117440","-2305843009213693952","-1729382256910270464","-1152921504606846976","-576460752303423488","0","576460752303423488","1152921504606846976","1729382256910270464","2305843009213693952","2882303761517117440","3458764513820540928","4035225266123964416","4611686018427387904","5188146770730811392","5764607523034234880","6341068275337658368","6917529027641081856","7493989779944505344","8070450532247928832","8646911284551352320"]
	}
}

variable "instance_hourly_cost" {
	type = map(number)
	description = "As of June 2025"
	default = {
		"t2.nano" = 0.0058
		"t2.micro" = 0.0116
		"t2.small" = 0.0230
		"t2.medium" = 0.0464
		"t2.large" = 0.0928
		"t2.xlarge" = 0.1856
		"t2.2xlarge" = 0.3712
		"c5ad.large" = 0.0860
		"c5ad.xlarge" = 0.1720
		"c5ad.2xlarge" = 0.3440
		"c5ad.4xlarge" = 0.6880
		"c5ad.8xlarge" = 1.3760
		"c5ad.12xlarge" = 2.0640
		"c5ad.16xlarge" = 2.7520
		"c5ad.24xlarge" = 4.1280
		"c6a.large" = 0.0765
		"c6a.xlarge" = 0.1530
		"c6a.2xlarge" = 0.3060
		"c6a.4xlarge" = 0.6120
		"c6a.8xlarge" = 1.2240
		"c6a.12xlarge" = 1.8360
		"c6a.16xlarge" = 2.4480
		"c6a.24xlarge" = 3.6720
		"c6a.32xlarge" = 4.8960
		"c6a.metal" = 7.3440
		"c6a.48xlarge" = 7.3440
		"c7g.medium" = 0.0363
		"c7g.large" = 0.0725
		"c7g.xlarge" = 0.1450
		"c7g.2xlarge" = 0.2900
		"c7g.4xlarge" = 0.5800
		"c7g.8xlarge" = 1.1600
		"c7g.12xlarge" = 1.7400
		"c7g.16xlarge" = 2.3200
		"c7g.metal" = 2.3200
		"c7gd.medium" = 0.0454
		"c7gd.large" = 0.0907
		"c7gd.xlarge" = 0.1814
		"c7gd.2xlarge" = 0.3629
		"c7gd.4xlarge" = 0.7258
		"c7gd.8xlarge" = 1.4515
		"c7gd.12xlarge" = 2.1773
		"c7gd.16xlarge" = 2.9030		
		"c7gd.metal" = 2.9030
		"c8g.medium" = 0.03988
		"c8g.large" = 0.07976
		"c8g.xlarge" = 0.15952
		"c8g.2xlarge" = 0.31904
		"c8g.4xlarge" = 0.63808
		"c8g.8xlarge" = 1.27616
		"c8g.12xlarge" = 1.91424
		"c8g.16xlarge" = 2.55232
		"c8gd.medium" = 0.04899
		"c8gd.large" = 0.09798
		"c8gd.xlarge" = 0.19596
		"c8gd.2xlarge" = 0.39192
		"c8gd.4xlarge" = 0.78384
		"c8gd.8xlarge" = 1.56768
		"c8gd.12xlarge" = 2.35152
		"c8gd.16xlarge" = 3.13536		
	}
}
variable "cpu_count_map" {
  type = map(number)
  default = {
	// Daemon
	"c6a.large"    = 2
    "c6a.xlarge"   = 4
    "c6a.2xlarge"  = 8
    "c6a.4xlarge"  = 16
	"c6a.8xlarge"  = 32
	"c7g.medium"   = 1
	"c7g.large"    = 2
    "c7g.xlarge"   = 4
    "c7g.2xlarge"  = 8
    "c7g.4xlarge"  = 16
    "c7g.8xlarge"  = 32
	"c8g.medium"   = 1
	"c8g.large"    = 2
    "c8g.xlarge"   = 4
    "c8g.2xlarge"  = 8
    "c8g.4xlarge"  = 16
    "c8g.8xlarge"  = 32
	// Cassandra
	"c5ad.large"    = 2
    "c5ad.xlarge"   = 4
    "c5ad.2xlarge"  = 8
    "c5ad.4xlarge"  = 16
    "c5ad.8xlarge"  = 32
    "c5ad.16xlarge" = 64
	"c7gd.large"    = 2
    "c7gd.xlarge"   = 4
    "c7gd.2xlarge"  = 8
    "c7gd.4xlarge"  = 16
    "c7gd.8xlarge"  = 32
    "c7gd.16xlarge" = 64
	"c8gd.large"    = 2
    "c8gd.xlarge"   = 4
    "c8gd.2xlarge"  = 8
    "c8gd.4xlarge"  = 16
    "c8gd.8xlarge"  = 32
    "c8gd.16xlarge" = 64
  }
}

variable "instance_memory_map" {
  type = map(number)
  description = "Instance memory in GiB"
  default = {
	// Daemon
	"c6a.large"    = 4
    "c6a.xlarge"   = 8
    "c6a.2xlarge"  = 16
    "c6a.4xlarge"  = 32
	"c6a.8xlarge"  = 64
	"c7g.medium"   = 2
	"c7g.large"    = 4
    "c7g.xlarge"   = 8
    "c7g.2xlarge"  = 16
    "c7g.4xlarge"  = 32
    "c7g.8xlarge"  = 64
	"c8g.medium"   = 2
	"c8g.large"    = 4
    "c8g.xlarge"   = 8
    "c8g.2xlarge"  = 16
    "c8g.4xlarge"  = 32
    "c8g.8xlarge"  = 64
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
	"c7gd.large"    = "nvme[0-9]n[0-9] 109.9G"
    "c7gd.xlarge"   = "nvme[0-9]n[0-9] 220.7G" # quick_lookup 23s, lsblk: cassandra data0 nvme1n1 220.7G, bastion /mnt/capi_log nvme1n1 10G
    "c7gd.2xlarge"  = "nvme[0-9]n[0-9] 441.4G" # Portfolio: intel writers=cpus 4-1870s writers=cpus*1.5 1881 , 8-890s
    "c7gd.4xlarge"  = "nvme[0-9]n[0-9] 884.8G"
    "c7gd.8xlarge"  = "nvme[0-9]n[0-9] 1.7T"
    "c7gd.16xlarge" = "nvme[0-9]n[0-9] 1.7T"
	"c8gd.large"    = "nvme[0-9]n[0-9] 109.9G"
    "c8gd.xlarge"   = "nvme[0-9]n[0-9] 220.7G"
    "c8gd.2xlarge"  = "nvme[0-9]n[0-9] 441.4G" # Same overall perf as c7, not worth the price, Cassandra load: c7g 70%, c8g 45%
    "c8gd.4xlarge"  = "nvme[0-9]n[0-9] 884.8G"
    "c8gd.8xlarge"  = "nvme[0-9]n[0-9] 1.7T"
    "c8gd.16xlarge" = "nvme[0-9]n[0-9] 1.7T"
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
	default     = "1:27.3.4.2-1"
}

variable "rabbitmq_server_version_amd64" {
	type        = string
	description = "Latest RabbitMQ server from RabbitMQ team"
	default     = "4.1.4-1"
}

variable "rabbitmq_erlang_version_arm64" {
	type        = string
	description = "Ideally, Erlang version should match amd64 releases, but RabbitMQ team is late with arm64 for some reason. Watch RabbitMQ team changing this sometimes as of 2024-2025: 1ubuntu4, 1ubuntu4.1, 1ubuntu4.2."
	default     = "1:25.3.2.8+dfsg-1ubuntu4"
}

variable "rabbitmq_server_version_arm64" {
	type        = string
	description = "Older RabbitMQ server, because newer versions require newer Erlang 27 not supported on arm64 by RabbitMQ team"
	default     = "3.13.7-1ubuntu1"
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
	default     = "3.7.0"
}

variable "daemon_thread_pool_factor" {
	type        = number
	description= "Worker threads per Daemon CPU: 1.0 very conservative, 2.5 pretty aggressive"
	default     = 3
}

variable "daemon_writer_workers" {
	type        = number
	description= "stick with 6 for now"
	default     = 12
}

locals {
	cassandra_hosts            = join(",", [ for i in range(var.number_of_cassandra_hosts) : format("10.5.0.%02s", i+11) ])
    cassandra_initial_tokens   = var.cassandra_initial_tokens_map[var.number_of_cassandra_hosts]
	rabbitmq_url               = join("",  ["amqp://", var.rabbitmq_user_name, ":", var.rabbitmq_user_pass, "@10.5.1.10/"])
    prometheus_node_targets    = join(",",concat( # "\'localhost:9100\',\'10.5.1.10:9100\'"
										["'localhost:9100'"], // bastion node exporter
										[ for i in range(var.number_of_cassandra_hosts) : format("'10.5.0.%02s:9100'", i+11) ], // cassandra node exporters
										[ for i in range(var.number_of_daemons) : format("'10.5.0.1%02s:9100'", i+1) ])) // daemon node expoters
    prometheus_jmx_targets     = join(",", [ for i in range(var.number_of_cassandra_hosts) : format("'10.5.0.%02s:7070'", i+11) ]) // cassandra JMX exporters
    prometheus_go_targets      = join(",", concat( ["'localhost:9200'"], [ for i in range(var.number_of_daemons) : format("'10.5.0.1%02s:9200'", i+1) ])) // webapi and daemon go exporters
	daemon_thread_pool_size    = ceil(var.cpu_count_map[var.daemon_instance_type] * var.daemon_thread_pool_factor )
	daemon_gomemlimit_gb       = ceil(var.instance_memory_map[var.daemon_instance_type] * 0.75 ) // Let daemon use half of RAM, GOGC=100 will probably take it to 70%, and we also need some memory to run Python
	webapi_gomemlimit_gb       = ceil(var.instance_memory_map[var.bastion_instance_type] / 2 )
} 

# Env variables TF_VAR_

variable "BASTION_ALLOWED_IPS" {
	type        = string
	description = "Comma-separated list of IP addresses and cidr blocks allowed to access bastion from the outside"
}

# Output


output "output_bastion_public_ip" {
  value = aws_eip.bastion_public_ip.public_ip
}

output "output_daemon_cpus" {
  value = format("%d cpus ($%f/hr)", var.cpu_count_map[var.daemon_instance_type], var.instance_hourly_cost[var.daemon_instance_type])
}

output "output_daemon_instances" {
  value = var.number_of_daemons
}

output "output_daemon_total_cpus" {
  value = var.cpu_count_map[var.daemon_instance_type]*var.number_of_daemons
}

output "output_cassandra_cpus" {
  value = format("%d cpus ($%f/hr)", var.cpu_count_map[var.cassandra_instance_type], var.instance_hourly_cost[var.cassandra_instance_type])
}

output "output_cassandra_hosts" {
  value = var.number_of_cassandra_hosts
}

output "output_cassandra_total_cpus" {
  value = var.cpu_count_map[var.cassandra_instance_type]*var.number_of_cassandra_hosts
}

output "output_daemon_thread_pool_size" {
  value = local.daemon_thread_pool_size
}

output "output_daemon_writer_workers" {
  value = var.daemon_writer_workers
}

output "output_daemon_writers_per_cassandra_cpu" {
  value = local.daemon_thread_pool_size * var.daemon_writer_workers * var.number_of_daemons / (var.cpu_count_map[var.cassandra_instance_type]*var.number_of_cassandra_hosts)
}

output "output_total_ec2_hourly_cost" {
  value = format("$%f/hr",var.instance_hourly_cost[var.cassandra_instance_type]*var.number_of_cassandra_hosts + var.instance_hourly_cost[var.daemon_instance_type]*var.number_of_daemons + var.instance_hourly_cost[var.bastion_instance_type])
}

output "output_vars_bastion_provisioner_static_vars" {
  value = local.bastion_provisioner_static_vars
}
output "output_vars_cassandra_provisioner_vars" {
  value = local.cassandra_provisioner_vars
}

output "output_vars_daemon_provisioner_vars" {
  value = local.daemon_provisioner_vars
}
