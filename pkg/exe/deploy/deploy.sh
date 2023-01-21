# Create all instances in one shot (54 s)
go run deploy.go create_instances bastion,daemon01,cass01,cass02,cass03,rabbitmq,prometheus

# Make sure they are available
go run deploy.go ping_instances bastion,daemon01,cass01,cass02,cass03,rabbitmq,prometheus

# Create sftp user
go run deploy.go create_instance_users bastion

# Allow these hosts to connect to data via sftp
go run deploy.go copy_private_keys bastion,daemon01

# Volumes: used only by bastion
#go run deploy.go create_volumes
# Attach volumes and make sftpuser owner (15s)
go run deploy.go attach_volumes bastion

# Upload all files in one shot (2 min)
go run deploy.go upload_files up_daemon_env_config,up_daemon_binary,up_webapi_env_config,up_webapi_binary,up_ui,up_test_in,up_test_out,up_test_cfg

# Setup all services except daemons (2 min)
go run deploy.go setup_services bastion,cass01,cass02,cass03,prometheus,rabbitmq

# Setup capidaemons (30 s)
go run deploy.go setup_services daemon01

# Check cassandra nodetool, all 3 shouldbe up, no exceptions thrown:
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment001_rsa -J 208.113.134.216 ubuntu@10.5.0.11 'nodetool status'

Check cassandra status - this should return no error
http://208.113.134.216:6543/ks

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

go run deploy.go download_files down_capi_out,down_capi_logs