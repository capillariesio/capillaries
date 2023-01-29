# Openstack environment

1. Make sure you have an Openstack project ready, with all OS_* variables defined in the project parameters file
2. Make sure you have created the key pair for SSH access to the Openstack instances, key pair name stored in `root_key_name`.
3. Make sure all configuration values in the project parameters file are up-to-date.
4. This guide assumes you have reserved a floating IP address 208.113.134.216 with your Openstack provider, this address will be assigned to the `bastion` instance and will be your door to all of your instances. Make sure you have set up this IP address for a jump host in your ssh configuration and assign it to `floating_ip_address` in your project parameters json file.

# Openstack networking and volumes

This step does not need to be performed often, assuming the Openstack provider does not charge for networking and volumes.

```
cd test/deploy
go run ../../pkg/exe/deploy/capideploy.go create_security_group
go run ../../pkg/exe/deploy/capideploy.go create_networking
go run ../../pkg/exe/deploy/capideploy.go create_volumes
```

# Build Capillaries binaries

Build binaries for Linux:

```
cd test/deploy
GOOS=linux GOARCH=amd64 go build -o ../../build/linux/amd64/capidaemon -ldflags="-s -w" ../../pkg/exe/daemon/capidaemon.go
gzip -f ../../build/linux/amd64/capidaemon
GOOS=linux GOARCH=amd64 go build -o ../../build/linux/amd64/capiwebapi -ldflags="-s -w" ../../pkg/exe/webapi/capiwebapi.go
gzip -f ../../build/linux/amd64/capiwebapi
GOOS=linux GOARCH=amd64 go build -o ../../build/linux/amd64/capitoolbelt -ldflags="-s -w" ../../pkg/exe/toolbelt/capitoolbelt.go
gzip -f ../../build/linux/amd64/capitoolbelt

pushd ../../ui
set CAPILLARIES_WEBAPI_URL=http://208.113.134.216
npm run build
popd
```

# Prepare test data

This command will populate /tmp/capi_in, /tmp/capi_cfg, /tmp/capi_out

```
cd test/code/lookup
./1_create_quicktest_data.sh
./create_bigtest_data.sh
cd test/code/tag_and_denormalize
./1_create_quicktest_data.sh
```

Deploy project will pick up the files to upload from there.

# Build test environment 

```
# Create all instances in one shot (1 min)
go run ../../pkg/exe/deploy/capideploy.go create_instances bastion,daemon01,daemon02,cass01,cass02,cass03,cass04,cass05,rabbitmq,prometheus

# Make sure they are available. If an instance is missing for too long, go to the provider console and reboot if needed (happens sometimes)
go run ../../pkg/exe/deploy/capideploy.go ping_instances bastion,daemon01,daemon02,cass01,cass02,cass03,cass04,cass05,rabbitmq,prometheus

# Create sftp user ((we assume the Openstack provider doesnot support multi-attach volumes, so we have to use sftp to read and write files)
go run ../../pkg/exe/deploy/capideploy.go create_instance_users bastion

# Allow these instances to connect to data via sftp
go run ../../pkg/exe/deploy/capideploy.go copy_private_keys bastion,daemon01,daemon02

# Attach volumes and make sftpuser owner (15 s)
go run ../../pkg/exe/deploy/capideploy.go attach_volumes bastion

# Upload all files in one shot (1 min). Make sure you have all binaries built before uploading them.
go run ../../pkg/exe/deploy/capideploy.go upload_files up_daemon_binary,up_daemon_env_config,up_webapi_env_config,up_webapi_binary,up_ui,up_toolbelt_env_config,up_toolbelt_binary,up_all_cfg,up_lookup_bigtest_in,up_lookup_bigtest_out,up_lookup_quicktest_in,up_lookup_quicktest_out,up_tag_and_denormalize_quicktest_in,up_tag_and_denormalize_quicktest_out,up_py_calc_quicktest_in,up_py_calc_quicktest_out

go run ../../pkg/exe/deploy/capideploy.go upload_files up_all_cfg
go run ../../pkg/exe/deploy/capideploy.go upload_files up_daemon_binary,up_daemon_env_config

# Setup all services (2 min)
go run ../../pkg/exe/deploy/capideploy.go setup_services bastion,cass01,cass02,cass03,cass04,cass05,prometheus,rabbitmq,daemon01,daemon02

# Check Cassandra with nodetool, all should be up (UN), no exceptions thrown:
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment001_rsa -J 208.113.134.216 ubuntu@10.5.0.11 'nodetool status'
````

Check Cassandra status - this Webapi call should return no error: `http://208.113.134.216:6543/ks`

Check RabbitMQ is functioning at `http://208.113.134.216:15672` (use RabbitMQ username/password from project parameters file)

# Run test processes

At `http://208.113.134.216`, create new runs:

| Field | Value |
|- | - |
| Keyspace | tag_and_denormalize_quicktest |
| Script URI | sftp://sftpuser@10.5.0.2/mnt/capi_cfg/tag_and_denormalize_quicktest/script.json |
| Script parameters URI | sftp://sftpuser@10.5.0.2/mnt/capi_cfg/tag_and_denormalize_quicktest/script_params_one_run.json |
| Start nodes |	read_tags,read_products |

or
```
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment001_rsa ubuntu@208.113.134.216 '~/bin/capitoolbelt start_run -script_file=sftp://sftpuser@10.5.0.2/mnt/capi_cfg/tag_and_denormalize_quicktest/script.json -params_file=sftp://sftpuser@10.5.0.2/mnt/capi_cfg/tag_and_denormalize_quicktest/script_params_one_run.json -keyspace=tag_and_denormalize_quicktest -start_nodes=read_tags,read_products'
```

| Field | Value |
|- | - |
| Keyspace | lookup_quicktest |
| Script URI | sftp://sftpuser@10.5.0.2/mnt/capi_cfg/lookup_quicktest/script.json |
| Script parameters URI | sftp://sftpuser@10.5.0.2/mnt/capi_cfg/lookup_quicktest/script_params_one_run.json |
| Start nodes |	read_orders,read_order_items |

or

ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment001_rsa ubuntu@208.113.134.216 '~/bin/capitoolbelt start_run -script_file=sftp://sftpuser@10.5.0.2/mnt/capi_cfg/lookup_quicktest/script.json -params_file=sftp://sftpuser@10.5.0.2/mnt/capi_cfg/lookup_quicktest/script_params_one_run.json -keyspace=lookup_quicktest -start_nodes=read_orders,read_order_items'

| Field | Value |
|- | - |
| Keyspace | lookup_bigtest |
| Script URI | sftp://sftpuser@10.5.0.2/mnt/capi_cfg/lookup_bigtest/script.json |
| Script parameters URI | sftp://sftpuser@10.5.0.2/mnt/capi_cfg/lookup_bigtest/script_params_one_run.json |
| Start nodes |	read_orders,read_order_items |

or
```
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment001_rsa ubuntu@208.113.134.216 '~/bin/capitoolbelt start_run -script_file=sftp://sftpuser@10.5.0.2/mnt/capi_cfg/lookup_bigtest/script.json -params_file=sftp://sftpuser@10.5.0.2/mnt/capi_cfg/lookup_bigtest/script_params_one_run.json -keyspace=lookup_bigtest -start_nodes=read_orders,read_order_items'

py_calc

ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment001_rsa ubuntu@208.113.134.216 '~/bin/capitoolbelt start_run -script_file=sftp://sftpuser@10.5.0.2/mnt/capi_cfg/py_calc_quicktest/script.json -params_file=sftp://sftpuser@10.5.0.2/mnt/capi_cfg/py_calc_quicktest/script_params_one_run.json -keyspace=py_calc_quicktest -start_nodes=read_order_items'

```

# Prometheus: watch instance load

| Metric | Prometheus screen |
|- | - |
| CPU usage % | `http://208.113.134.216:9090/graph?g0.expr=(1%20-%20avg(irate(node_cpu_seconds_total%7Bmode%3D%22idle%22%7D%5B10m%5D))%20by%20(instance))%20*%20100&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=15m` |
| RAM usage % | `http://208.113.134.216:9090/graph?g0.expr=100%20*%20(1%20-%20((avg_over_time(node_memory_MemFree_bytes%5B10m%5D)%20%2B%20avg_over_time(node_memory_Cached_bytes%5B10m%5D)%20%2B%20avg_over_time(node_memory_Buffers_bytes%5B10m%5D))%20%2F%20avg_over_time(node_memory_MemTotal_bytes%5B10m%5D)))&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=15m` |
| Disk usage % | `http://208.113.134.216:9090/graph?g0.expr=100%20-%20((node_filesystem_avail_bytes%7Bmountpoint%3D%22%2F%22%2Cfstype!%3D%22rootfs%22%7D%20*%20100)%2Fnode_filesystem_size_bytes%7Bmountpoint%3D%22%2F%22%2Cfstype!%3D%22rootfs%22%7D)&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=15m` |

# Results

Download all results from capi_out:

```
go run ../../pkg/exe/deploy/capideploy.go download_files down_capi_out
```

# Logging

See consolidated log from all capidaemons:

```
go run ../../pkg/exe/deploy/capideploy.go download_files down_capi_logs
```

# Delete test environment

```
go run ../../pkg/exe/deploy/capideploy.go delete_instances all
```

and, if needed:

```
go run ../../pkg/exe/deploy/capideploy.go delete_volumes
```




