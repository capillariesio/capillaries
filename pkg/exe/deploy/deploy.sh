# Create all instances in one shot (54 s)
go run capideploy.go create_instances bastion,daemon01,daemon02,cass01,cass02,cass03,rabbitmq,prometheus

# Make sure they are available. If an instance is missing for too long, go to the provider console and reboot if needed (happens sometimes)
go run capideploy.go ping_instances bastion,daemon01,daemon02,cass01,cass02,cass03,rabbitmq,prometheus

# Create sftp user
go run capideploy.go create_instance_users bastion

# Allow these hosts to connect to data via sftp
go run capideploy.go copy_private_keys bastion,daemon01,daemon02

# Volumes: used only by bastion
#go run capideploy.go create_volumes
# Attach volumes and make sftpuser owner (15s)
go run capideploy.go attach_volumes bastion

# Upload all files in one shot (2 min). Make sure you have all binaries built and ready before uploading them.
go run capideploy.go upload_files up_daemon_env_config,up_daemon_binary,up_webapi_env_config,up_webapi_binary,up_ui,up_test_in,up_test_out,up_test_cfg

# Setup all services except daemons (2 min)
go run capideploy.go setup_services bastion,cass01,cass02,cass03,prometheus,rabbitmq

# Setup capidaemons (30 s)
go run capideploy.go setup_services daemon01,daemon02

# Check cassandra nodetool, all 3 shouldbe up, no exceptions thrown:
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment001_rsa -J 208.113.134.216 ubuntu@10.5.0.11 'nodetool status'

# Check cassandra status - this should return no error
curl http://208.113.134.216:6543/ks

CPU usage %
http://208.113.134.216:9090/graph?g0.expr=(1%20-%20avg(irate(node_cpu_seconds_total%7Bmode%3D%22idle%22%7D%5B10m%5D))%20by%20(instance))%20*%20100&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=15m
RAM usage%
http://208.113.134.216:9090/graph?g0.expr=100%20*%20(1%20-%20((avg_over_time(node_memory_MemFree_bytes%5B10m%5D)%20%2B%20avg_over_time(node_memory_Cached_bytes%5B10m%5D)%20%2B%20avg_over_time(node_memory_Buffers_bytes%5B10m%5D))%20%2F%20avg_over_time(node_memory_MemTotal_bytes%5B10m%5D)))&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=15m
Disk usage %
http://208.113.134.216:9090/graph?g0.expr=100%20-%20((node_filesystem_avail_bytes%7Bmountpoint%3D%22%2F%22%2Cfstype!%3D%22rootfs%22%7D%20*%20100)%2Fnode_filesystem_size_bytes%7Bmountpoint%3D%22%2F%22%2Cfstype!%3D%22rootfs%22%7D)&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=15m

RabbitMQ
http://208.113.134.216:15672/#/queues


# http://208.113.134.216/
# tag_and_denormalize
# sftp://sftpuser@10.5.0.2/mnt/capi_cfg/tag_and_denormalize/script.json
# sftp://sftpuser@10.5.0.2/mnt/capi_cfg/tag_and_denormalize/script_params_one_run.json
# read_tags,read_products

# lookup
# sftp://sftpuser@10.5.0.2/mnt/capi_cfg/lookup/script.json
# sftp://sftpuser@10.5.0.2/mnt/capi_cfg/lookup/script_params_one_run.json
# read_orders,read_order_items

go run capideploy.go download_files down_capi_out,down_capi_logs

# sudo apt update
# sudo apt-get install iperf3
# Server: sudo iperf3 -s -p 80
# Client: perf3 -c 10.5.0.2 -p 80
iperf3 -c10.5.0.2 -p 80
Connecting to host 10.5.0.2, port 80
[  5] local 10.5.0.101 port 47716 connected to 10.5.0.2 port 80
[ ID] Interval           Transfer     Bitrate         Retr  Cwnd
[  5]   0.00-1.00   sec   922 MBytes  7.73 Gbits/sec  258   2.10 MBytes
[  5]   1.00-2.00   sec   810 MBytes  6.79 Gbits/sec   43   2.47 MBytes
[  5]   2.00-3.00   sec   780 MBytes  6.55 Gbits/sec   93   1.87 MBytes
[  5]   3.00-4.00   sec   684 MBytes  5.73 Gbits/sec   18   2.20 MBytes
[  5]   4.00-5.00   sec   701 MBytes  5.88 Gbits/sec   96   1.96 MBytes
[  5]   5.00-6.00   sec   648 MBytes  5.43 Gbits/sec   28   2.48 MBytes
[  5]   6.00-7.00   sec   724 MBytes  6.07 Gbits/sec   91   1.94 MBytes
[  5]   7.00-8.00   sec   836 MBytes  7.02 Gbits/sec   57   2.39 MBytes
[  5]   8.00-9.00   sec   751 MBytes  6.30 Gbits/sec   25   2.43 MBytes
[  5]   9.00-10.00  sec   679 MBytes  5.69 Gbits/sec   54   2.25 MBytes
- - - - - - - - - - - - - - - - - - - - - - - - -
[ ID] Interval           Transfer     Bitrate         Retr
[  5]   0.00-10.00  sec  7.36 GBytes  6.32 Gbits/sec  763             sender
[  5]   0.00-10.00  sec  7.36 GBytes  6.32 Gbits/sec                  receiver

