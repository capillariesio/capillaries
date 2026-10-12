output "_00_output_vars_bastion_provisioner_static_vars" {
  value = local.bastion_provisioner_static_vars
}
output "_01_output_vars_cassandra_provisioner_vars" {
  value = local.cassandra_provisioner_vars
}

output "_02_output_vars_daemon_provisioner_vars" {
  value = local.daemon_provisioner_vars
}

output "_04_output_bastion_public_ip" {
  value = aws_eip.bastion_public_ip.public_ip
}

output "_10_output_daemon_instances" {
  value = var.number_of_daemons
}

output "_12_output_daemon_cpus" {
  value = format("%d cpus ($%f/hr)", var.cpu_count_map[var.daemon_instance_type], var.instance_hourly_cost[var.daemon_instance_type])
}

output "_13_output_daemon_total_cpus" {
  value = var.cpu_count_map[var.daemon_instance_type]*var.number_of_daemons
}

output "_14_output_daemon_memory_gb" {
  value = format("%d GiB", var.instance_memory_map[var.daemon_instance_type])
}

output "_15_output_daemon_gomemlimit_ratio" {
  value = var.daemon_instance_gomemlimit_ratio_map[var.daemon_instance_type]
}

output "_16_output_daemon_gomemlimit_mb" {
  value = format("%d MiB", local.daemon_gomemlimit_mb)
}

output "_17_output_daemon_gogc" {
  value = var.daemon_gogc
}

output "_18_output_daemon_thread_pool_size" {
  value = local.daemon_thread_pool_size
}

output "_19_output_daemon_writer_workers" {
  value = var.daemon_writer_workers
}

output "_1A_output_daemon_lookup_key_workers" {
  value = var.daemon_lookup_key_workers
}

output "_1B_output_daemon_lookup_rowid_workers" {
  value = var.daemon_lookup_rowid_workers
}

output "_1C_output_daemon_writers_per_cassandra_cpu" {
  value = local.daemon_thread_pool_size * var.daemon_writer_workers * var.number_of_daemons / (var.cpu_count_map[var.cassandra_instance_type]*var.number_of_cassandra_hosts)
}

output "_20_output_cassandra_hosts" {
  value = var.number_of_cassandra_hosts
}

output "_21_output_cassandra_cpus" {
  value = format("%d cpus ($%f/hr)", var.cpu_count_map[var.cassandra_instance_type], var.instance_hourly_cost[var.cassandra_instance_type])
}

output "_23_output_cassandra_total_cpus" {
  value = var.cpu_count_map[var.cassandra_instance_type]*var.number_of_cassandra_hosts
}

output "_30_output_total_ec2_hourly_cost" {
  value = format("$%f/hr",var.instance_hourly_cost[var.cassandra_instance_type]*var.number_of_cassandra_hosts + var.instance_hourly_cost[var.daemon_instance_type]*var.number_of_daemons + var.instance_hourly_cost[var.bastion_instance_type])
}

# output "_40_test" {
#   value = local.prometheus_node_exporter_url
# }

