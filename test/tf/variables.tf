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
    default     = 8
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

# 4 x c7g.large, 5 writers, threadpool 3
# accounts 6900
# txns 7700 (daemons 20% cpu, 35-45s per batch)

# 8 writers

#1_read_period_holdings: Load holdings from parquet
# period holdings 8300 write, same idx_period holdings

#1_read_txns: Load txns from parquet
# txns 20-30s per batch, load w12000, daemon instance cpu 30%

#2_account_txns_outer: For each account, merge all txns into single json string
# when building accnt_txns, txns read units at 8900
# accounts_txns_00001 w460, txns r8800, each batch takes 30s
# Then CPU drops to 20%, then 10%

# 2_account_period_holdings_outer: For each account, merge all holdings into single json string
# when building period_hoding_outer write units 320, period_holdings_00001 r8200, each batch takes 10s
# period_hodings_0001 w5900

# 3_build_account_period_activity: For each account, merge holdings and txns
# error running node 3_build_account_period_activity of type table_lookup_table in the script [s3://capillaries-testbucket/capi_cfg/portfolio_bigtest/script.json]: [cannot close iterator after 5 attempts and 6200ms, still getting timeouts, query:SELECT rowid, token(rowid), account_id, txns_json FROM portfolio_bigtest_cloud.account_txns_00001 WHERE token(rowid) >= ? AND token(rowid) <= ? LIMIT 1000;, dberror:Operation timed out - received only 0 responses.]
# Enough read bandwidth, dunno why the problem

# Writes to account_period_activity_00001 are heavily throttled
# error running node 3_build_account_period_activity of type table_lookup_table in the script [s3://capillaries-testbucket/capi_cfg/portfolio_bigtest/script.json]: [cannot insert data record [INSERT INTO portfolio_bigtest_cloud.account_period_activity_00001 ( rowid, batch_idx, holdings_json, account_id, txns_json ) VALUES ( ?, ?, ?, ?, ? ) IF NOT EXISTS;]: cannot write to data table account_period_activity_00001 after 5 attempts and 6200ms, still getting timeouts: Operation timed out - received only 0 responses.; table inserter detected slow db insertion rate 2 records/s, wrote 6 records out of 6]