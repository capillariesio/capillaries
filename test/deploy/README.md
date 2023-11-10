# Working with Capillaries deploy tool

Capillaries [Deploy tool](../../doc/glossary.md#deploy-tool) can provision complete Capillaries cloud environment in public/private clouds that support [Openstack API](https://www.openstack.org) or in AWS.

`test/deploy` directory contains sample project template (sampledeployment.jsonnet) and sample project (sampledeployment.json) used by [Deploy tool](../../doc/glossary.md#deploy-tool).

For troubleshooting, add `-verbose` argument to your deploy tool command line.

## The goal

On the diagram below, each rectangle represents a VPS instance that performs some specific task. Please note there are multiple Cassandra node instances (we need a Cassadra cluster) and multiple instances running Capillaries Daemon (this is how Capillaries scales out). Prometheus and RabbitMQ are not bottlenecks, so one instance for each is enough. Bastion instance is the gateway to the private network where the deployment is hosted, it:
- runs a reverse proxy so users can access Prometheus and RabbitMQ web consoles
- runs Capillaries Webapi and UI so users can start Capillaries runs and see their execution status
- has Capillaries Toolbelt installed so SSH users an start runs from the command line
- works as a jump host so users have SSH access to other instances in the private network
- accumulates Capillaries Daemon logs in /var/log/capillaries using rsyslog

Capillaries configuration scripts and in/out data are stored on separate volumes. In current test implementation, Bastion instances mounts them as /mnt/capi_cfg,/mnt/capi_in, /mnt/capi_out and gives Capillaries Daemons SFTP and NFS access to them. 

![Public cloud deployment](./doc/cloud-deployment.svg)

## Deployment project template (`*.jsonnet`) and deployment project (`*.json`) files

Capideploy tool uses deployment project file (see sample `sampledeployment.json`) to:
- configure creation of Openstack/AWS objects like instances and volumes and track status of those objects locally
- push Capillaries data and binaries to created Openstack/AWS deployment
- clean Openstack/AWS deployment

Deployment project files contain description and status of each instance. When there are a lot of instances that perform the same tesk (like Cassandra nodes or instances running Capillaries [Daemon](../../doc/glossary.md#daemon)) which makes them pretty redundant. To avoid creating repetitive configurations manually, use [jsonnet](https://jsonnet.org) templates like `sampledeployment.jsonnet`. Before deploying, make sure that you have generated a deployment project `*.json` file from the `*.jsonnet` template, and, under normal circumstances, avoid manual changes in your `*.json` file. Tweak `*.jsonnet` file and regenerate `*.json` instead, using jsonnet interpreter of your choice. Feel free to manually tweak `*.json` file if you really think you know what you are doing.

## Before deployment

1. Install [jq](https://jqlang.github.io/jq/). Adding jq to the list of requirements was not an easy decision, but without it, [start_cluster.sh](./start_cluster.sh) script that has to read configuration from the deployment project file would be unnecessary error-prone.

2. Make sure you have created the key pair for SSH access to the Openstack/AWS instances, key pair name stored in `root_key_name` in the project file. Through this document, we will be assuming the key pair is stored in `~/.ssh/` and the private key file this kind of name:  
`sampledeployment002_rsa`.

3. If you want to use SFTP (instead of or along with NFS) for file sharing make sure all SFTP key files used referenced in deployment project `sampledeployment.json` are present.

4. Make sure all environment variables storing Capideploy and Openstack/AWS settings are set. For non-production environments, you may want to keep them in a separate private file and activate before deploying `source ~/sampledeployment.rc`:

```
# capideploy settings

export CAPIDEPLOY_SSH_USER=ubuntu
export CAPIDEPLOY_SSH_PRIVATE_KEY_PASS=""
export CAPIDEPLOY_SFTP_USER=...
export CAPIDEPLOY_RABBITMQ_ADMIN_NAME=...
export CAPIDEPLOY_RABBITMQ_ADMIN_PASS=...
export CAPIDEPLOY_RABBITMQ_USER_NAME=...
export CAPIDEPLOY_RABBITMQ_USER_PASS=...

# OpenStack settings

# Example 
export OS_AUTH_URL=https://us-central-1.genesishosting.com:5000/v3
export OS_PROJECT_ID=7abdc4...
export OS_PROJECT_NAME="myProject"
export OS_USER_DOMAIN_NAME="myDomain"
export OS_PROJECT_DOMAIN_ID=8b2a...
export OS_USERNAME="myAdmin"
export OS_PASSWORD="j7F04F..."
export OS_REGION_NAME="us-central-1"
export OS_INTERFACE=public
export OS_IDENTITY_API_VERSION=3

# Example 003
export OS_AUTH_URL="https://auth.cloud.ovh.net/v3"
export OS_IDENTITY_API_VERSION=3
export OS_USER_DOMAIN_NAME="default"
export OS_TENANT_ID="d6b34..."
export OS_TENANT_NAME="743856365..."
export OS_USERNAME="user-h7FGd43..."
export OS_PASSWORD='hS56Shr...'
export OS_REGION_NAME="BHS5"
export OS_PROJECT_ID=d6b34...
export OS_PROJECT_NAME="743856365..."
export OS_INTERFACE="public"

# Example 004
unset OS_TENANT_ID
unset OS_TENANT_NAME
export OS_AUTH_URL=https://api.pub1.infomaniak.cloud/identity
export OS_PROJECT_ID=67da20...
export OS_PROJECT_NAME="PCP-..."
export OS_USER_DOMAIN_NAME="Default"
export OS_PROJECT_DOMAIN_ID="default"
export OS_USERNAME="PCU-..."
export OS_PASSWORD=s0me$ecurepass...
export OS_REGION_NAME="dc3-a"
export OS_INTERFACE=public
export OS_IDENTITY_API_VERSION=3

# AWS settings

export AWS_ACCESS_KEY_ID=A...
export AWS_SECRET_ACCESS_KEY=...
export AWS_DEFAULT_REGION=us-east-1


```

5. Optionally, build deploy tool so you do not have to run `go run ../../build/capideploy.exe` every time and use `$capideploy` shortcut instead (this is a WSL example):

```
go build -o ../../build/capideploy.exe -ldflags="-s -w" ../../pkg/exe/deploy/capideploy.go
chmod 755 ../../build/capideploy.exe
export capideploy=../../build/capideploy.exe
```

From now on, this doc assumes `$capideploy` is present and functional.

6. Prepare Capillaries binaries (build/linux/amd64) and data (/tmp/capi_in, /tmp/capi_cfg, /tmp/capi_out):

```
$capideploy build_artifacts -prj=sampledeployment.json
```

7. Keep in mind that running deploy tool with `-verbose` parameter can be useful for troubleshooting.

## Deploy

```
# Reserve a floating IP address, it will be assigned to the bastion instance
# and will be your gateway to all of your instances:

$capideploy create_floating_ip -prj=sampledeployment.json

# If successful, create_floating_ip command will ask you to:
# - update your ~/.ssh/config with a new jumphost (this is by [start_cluster.sh](./start_cluster.sh), see below)
# - to use BASTION_IP environment variable when running commands against your deployment

# Openstack networking and volumes

$capideploy create_networking -prj=sampledeployment.json;
$capideploy create_security_groups -prj=sampledeployment.json;
$capideploy create_volumes '*' -prj=sampledeployment.json;

# Create all instances in one shot

$capideploy create_instances '*' -prj=sampledeployment.json

# Make sure we can actually login to each instance. If an instance is
# missing for too long, go to the provider console/logs for details

until $capideploy ping_instances '*' -prj=sampledeployment.json; do echo "Ping failed, wait..."; sleep 10; done

# Install all pre-requisite software

$capideploy install_services '*' -prj=sampledeployment.json

# Create sftp user on bastion host if needed and 
# allow these instances to connect to data via sftp

$capideploy create_instance_users bastion -prj=sampledeployment.json
$capideploy copy_private_keys 'bastion,daemon*' -prj=sampledeployment.json

# Attach bastion (and Cassandra, if needed) volumes,
# make ssh_user (or sftp_user, if you use sftp instead of nfs) owner

$capideploy attach_volumes '*' -prj=sampledeployment.json

# Now it's a good time to start Cassandra cluster in a SEPARATE shell session (that has `CAPIDEPLOY_*` environment variables set, see above). After strating it, and letting it run in parallel, you continue running command in the original shell session. If you run it with `parallel` parameter, make sure all Cassandra nodes are declared as seed in CASSANDRA_SEEDS, this will help avoid slow bootstrapping process. If, for some reason, this approach does not work for your Cassandra setup, use `sequential` parameter.

./start_cluster.sh parallel sampledeployment.json

# Upload binaries and their configs. Make sure you have all binaries and test data built before uploading them (see above).

$capideploy upload_files up_daemon_binary,up_daemon_env_config -prj=sampledeployment.json;
$capideploy upload_files up_webapi_binary,up_webapi_env_config -prj=sampledeployment.json;
$capideploy upload_files up_ui -prj=sampledeployment.json;
$capideploy upload_files up_toolbelt_binary,up_toolbelt_env_config -prj=sampledeployment.json;
$capideploy upload_files up_capiparquet_binary -prj=sampledeployment.json;
$capideploy upload_files up_diff_scripts -prj=sampledeployment.json;

# Upload test files (pick those that you need)

$capideploy upload_files up_all_cfg -prj=sampledeployment.json;
$capideploy upload_files up_portfolio_bigtest_in,up_portfolio_bigtest_out -prj=sampledeployment.json;
$capideploy upload_files up_lookup_bigtest_in,up_lookup_bigtest_out -prj=sampledeployment.json;
$capideploy upload_files up_lookup_quicktest_in,up_lookup_quicktest_out -prj=sampledeployment.json;
$capideploy upload_files up_tag_and_denormalize_quicktest_in,up_tag_and_denormalize_quicktest_out -prj=sampledeployment.json;
$capideploy upload_files up_py_calc_quicktest_in,up_py_calc_quicktest_out -prj=sampledeployment.json;
$capideploy upload_files up_portfolio_quicktest_in,up_portfolio_quicktest_out -prj=sampledeployment.json;

# Configure all services except Cassandra (which requires extra care), bastion first (it configs NFS)

$capideploy config_services bastion -prj=sampledeployment.json
$capideploy config_services 'rabbitmq,prometheus,daemon*' -prj=sampledeployment.json
```

## Monitoring test environment

RabbitMQ console (use RabbitMQ username/password from the project parameter file): `http://$BASTION_IP:15672`

Resource usage:

| Metric | Prometheus screen |
|- | - |
| CPU usage % | `http://$BASTION_IP:9090/graph?g0.expr=(1%20-%20avg(irate(node_cpu_seconds_total%7Bmode%3D%22idle%22%7D%5B10m%5D))%20by%20(instance))%20*%20100&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=15m` |
| RAM usage % | `http://$BASTION_IP:9090/graph?g0.expr=100%20*%20(1%20-%20((avg_over_time(node_memory_MemFree_bytes%5B10m%5D)%20%2B%20avg_over_time(node_memory_Cached_bytes%5B10m%5D)%20%2B%20avg_over_time(node_memory_Buffers_bytes%5B10m%5D))%20%2F%20avg_over_time(node_memory_MemTotal_bytes%5B10m%5D)))&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=15m` |
| Disk usage % | `http://$BASTION_IP:9090/graph?g0.expr=100%20-%20((node_filesystem_avail_bytes%7Bmountpoint%3D%22%2F%22%2Cfstype!%3D%22rootfs%22%7D%20*%20100)%2Fnode_filesystem_size_bytes%7Bmountpoint%3D%22%2F%22%2Cfstype!%3D%22rootfs%22%7D)&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=15m` |
| Cassandra writes: count/s and latency | `http://$BASTION_IP:9090/graph?g0.expr=sum(irate(cassandra_table_operation_latency_seconds_count%7Bkeyspace!%3D"system"%2Coperation%3D"write"%7D%5B1m%5D))&g0.tab=0&g0.stacked=0&g0.show_exemplars=1&g0.range_input=15m&g1.expr=avg(cassandra_table_operation_latency_seconds%7Bkeyspace!%3D"system"%2Coperation%3D"write"%7D)&g1.tab=0&g1.stacked=0&g1.show_exemplars=0&g1.range_input=15m` |

Consolidated [Daemon](../../doc/glossary.md#daemon) log from all Daemon instances:

```
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment002_rsa ubuntu@$BASTION_IP
less /var/log/capidaemon/capidaemon.log
```

## Run test processes

Start runs either using [Webapi](../../doc/glossary.md#webapi) at `http://$BASTION_IP` or using [Toolbelt](../../doc/glossary.md#toolbelt) via SSH.

### lookup_quicktest

| Field | Value |
|- | - |
| Keyspace | lookup_quicktest |
| Script URI | /mnt/capi_cfg/lookup_quicktest/script.json |
| Script parameters URI | /mnt/capi_cfg/lookup_quicktest/script_params_one_run.json |
| Start nodes |	read_orders,read_order_items |

or

```
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment002_rsa ubuntu@$BASTION_IP '~/bin/capitoolbelt start_run -script_file=/mnt/capi_cfg/lookup_quicktest/script.json -params_file=/mnt/capi_cfg/lookup_quicktest/script_params_one_run.json -keyspace=lookup_quicktest -start_nodes=read_orders,read_order_items'
```

### lookup_bigtest

| Field | Value |
|- | - |
| Keyspace | lookup_bigtest |
| Script URI | /mnt/capi_cfg/lookup_bigtest/script_parquet.json |
| Script parameters URI | /mnt/capi_cfg/lookup_bigtest/script_params_one_run.json |
| Start nodes |	read_orders,read_order_items |

or

```
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment002_rsa ubuntu@$BASTION_IP '~/bin/capitoolbelt start_run -script_file=/mnt/capi_cfg/lookup_bigtest/script_parquet.json -params_file=/mnt/capi_cfg/lookup_bigtest/script_params_one_run.json -keyspace=lookup_bigtest -start_nodes=read_orders,read_order_items'
```

### py_calc_quicktest

| Field | Value |
|- | - |
| Keyspace | py_calc_bigtest |
| Script URI | /mnt/capi_cfg/py_calc_bigtest/script.json |
| Script parameters URI | /mnt/capi_cfg/py_calc_bigtest/script_params.json |
| Start nodes |	read_order_items |

or

```
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment002_rsa ubuntu@$BASTION_IP '~/bin/capitoolbelt start_run -script_file=/mnt/capi_cfg/py_calc_quicktest/script.json -params_file=/mnt/capi_cfg/py_calc_quicktest/script_params.json -keyspace=py_calc_quicktest -start_nodes=read_order_items'
```

### tag_and_denormalize_quicktest

| Field | Value |
|- | - |
| Keyspace | tag_and_denormalize_quicktest |
| Script URI | /mnt/capi_cfg/tag_and_denormalize_quicktest/script.json |
| Script parameters URI | /mnt/capi_cfg/tag_and_denormalize_quicktest/script_params_one_run.json |
| Start nodes |	read_tags,read_products |

or

```
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment002_rsa ubuntu@$BASTION_IP '~/bin/capitoolbelt start_run -script_file=/mnt/capi_cfg/tag_and_denormalize_quicktest/script.json -params_file=/mnt/capi_cfg/tag_and_denormalize_quicktest/script_params_one_run.json -keyspace=tag_and_denormalize_quicktest -start_nodes=read_tags,read_products'
```


### portfolio_quicktest

| Field | Value |
|- | - |
| Keyspace | portfolio_quicktest |
| Script URI | /mnt/capi_cfg/portfolio_quicktest/script.json |
| Script parameters URI | /mnt/capi_cfg/portfolio_quicktest/script_params.json |
| Start nodes |	1_read_accounts,1_read_txns,1_read_period_holdings |

or

```
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment002_rsa ubuntu@$BASTION_IP '~/bin/capitoolbelt start_run -script_file=/mnt/capi_cfg/portfolio_quicktest/script.json -params_file=/mnt/capi_cfg/portfolio_quicktest/script_params.json -keyspace=portfolio_quicktest -start_nodes=1_read_accounts,1_read_txns,1_read_period_holdings'
```

### portfolio_bigtest

| Field | Value |
|- | - |
| Keyspace | portfolio_bigtest |
| Script URI | /mnt/capi_cfg/portfolio_bigtest/script.json |
| Script parameters URI | /mnt/capi_cfg/portfolio_bigtest/script_params.json |
| Start nodes |	1_read_accounts,1_read_txns,1_read_period_holdings |

or

```
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment002_rsa ubuntu@$BASTION_IP '~/bin/capitoolbelt start_run -script_file=/mnt/capi_cfg/portfolio_bigtest/script.json -params_file=/mnt/capi_cfg/portfolio_bigtest/script_params.json -keyspace=portfolio_bigtest -start_nodes=1_read_accounts,1_read_txns,1_read_period_holdings'
```

## Results

# capi_out

Download all results from capi_out (may take a while):

```
$capideploy download_files down_capi_out -prj=sampledeployment.json
```

Alternatively, verify results against the baseline remotely:

```
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment002_rsa ubuntu@$BASTION_IP '~/bin/compare_results_lookup_quicktest.sh'
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment002_rsa ubuntu@$BASTION_IP '~/bin/compare_results_lookup_bigtest_parquet.sh'
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment002_rsa ubuntu@$BASTION_IP '~/bin/compare_results_portfolio_quicktest.sh'
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment002_rsa ubuntu@$BASTION_IP '~/bin/compare_results_portfolio_bigtest_parquet.sh'
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment002_rsa ubuntu@$BASTION_IP '~/bin/compare_results_py_calc_quicktest.sh'
ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment002_rsa ubuntu@$BASTION_IP '~/bin/compare_results_tag_and_denormalize.sh'
```

# Logs

Download consolidated Daemon log (may be large, as debug logging is on by default):

```
$capideploy download_files down_capi_logs -prj=sampledeployment.json
```

Alternatively, check it out on bastion:

```
less /var/log/capidaemon/capidaemon.log
```


## Clear test environment

```
# Delete instances. Keep in mind that instances may be in a `deleting` state
# even after this command is complete, check your cloud provider console to verify.

$capideploy delete_instances '*' -prj=sampledeployment.json

# Delete volumes, networking, security groups and floating ip

$capideploy delete_volumes '*' -prj=sampledeployment.json;
$capideploy delete_security_groups -prj=sampledeployment.json;
$capideploy delete_networking -prj=sampledeployment.json;
$capideploy delete_floating_ip -prj=sampledeployment.json;
```

## Q&A

### Openstack/AWS environment variables

Q. The list of `OS_*` variables changes from one Openstack provider to another. Why?

A. The choice of Openstack variables required for authentication is up to the provider. Some providers may require more variables than other.

### Service: install vs. config

Q. Why having separate sets of commands for software installation and configuration?

A. Installing all Capillaries pre-requisites is a one-time job and can take a while. Configuring services is a job that:
- usually takes less less time to execute 
- can be executed multiple times per instance (like re-configuring Daemon thread parameters)
- may be required to be executed in some specific order (like adding nodes to Cassandra cluster)

### SFTP vs. NFS

Q. Sample deployment project mentions SFTP, but doesn't seem to use it. Why?

A. Just to demonstrate both approaches. Reading data via SFTP would require a lot of computing power on the bastion side (encription and compression), so the decision is to use NFS for all data: capi_cfg, capi_in, capi_out.
To use SFTP instead of NFS for reading configs, make according changes in the `*.jsonnet` file:
- set up_all_cfg.after.env.MOUNT_POINT_CFG to `'sftp://{CAPIDEPLOY_SFTP_USER}@' + internal_bastion_ip + '/mnt/capi_cfg'`
- set bastion.volumes.owner to `"{CAPIDEPLOY_SFTP_USER}"`
- remove `/mnt/capi_cfg` from all `NFS_DIRS` values
and make sure you use `sftp://sftpuser@10.5.0.10/mnt/capi_cfg` (instead of `/mnt/capi_cfg`) prefixes for script and script parameter files when starting a test process (either in WebUI or in the command line).

### Cassandra initial_token and num_tokens

Q. Sample project uses `num_tokens`=1 and predefined `initial_token` in cassandra.yaml. But I want to use multiple vnodes and I want my add/remove Cassandra nodes dynamically.

A. This example works well when you need to quickly provision an environment with a predefined number of Cassandra nodes in the cluster with zero redundancy and configure it for maximum uniform distribution. If you want more flexibility:
- disable `num_tokens` override in sh/cassandra.config.sh
- disable `allocate_tokens_for_local_replication_factor` override in sh/cassandra.config.sh
- do not specify `INITIAL_TOKEN` variable in the deployment project
- start Cassandra cluster without using [start_cluster.sh](./start_cluster.sh) script

### Non-Openstack clouds

Q. Does Deploy tool work with clouds that do not support Openstack/AWS? Azure, GCP?

A. Starting Sep 2023, deploy tool supportd seployment to AWS. No support for Azure or GCP.

### Why should I use another custom deploy tool?

Q. I am familiar with widely used infrastructure provisioning tools (Ansible, Saltstack etc). Can I use them to deploy Capillaries components instead of using Capillaries Deploy tool?

A. Absolutely. Capillaries Deploy tool was created to serve only one goal: to demonstrate that production-scale Capillaries deployment can be provisioned very quickly (within a few minutes) without using complex third-party software.
