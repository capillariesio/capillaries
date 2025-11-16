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
