# Working with Capillaries deploy tool

Capillaries [Deploy tool](../../doc/glossary.md#deploy-tool) can provision complete Capillaries cloud environment in public/private clouds that support Openstack API.

`test/deploy` directory contains two sample projects (capideploy_project_dreamhost.json and capideploy_project_genesis.json) used by [Deploy tool](../../doc/glossary.md#deploy-tool). Sensitive and repetitive configuration can be stored in project parameter files (capideploy_project_params_dreamhost.json and capideploy_project_params_genesis.json), andit's a good idea to store parameter files at somewhat secure location (like user home dir).

For troubleshooting, add `-verbose` argument to your deploy tool command line.

## Openstack environment

1. Make sure you have an Openstack project ready, with all `OS_*` variables defined in the project parameter file. Usually, Openstack cloud provider generates a shell script that sets all those variables for you. Just manually copy those values to the parameter file, one by one.
2. Make sure you have created the key pair for SSH access to the Openstack instances, key pair name stored in `root_key_name` in the project file.
3. Make sure all configuration values in the project parameters file are up-to-date. Some paramaters, like `sftp_user_private_key` may contain multi-line content - make sure you replace all line endings with `\n`.
4. This guide assumes you have reserved a floating IP address with your Openstack provider, this address will be assigned to the `bastion` instance and will be your gateway to all of your instances. Make sure `floating_ip_address` in your project parameter file is set to this floating IP address. You may want to use bastion instance as an SSH  jumphost, make sure you have set up this IP address for a jump host in `~/.ssh/config` file:

```
Host <dreamhost_bastion_ip>
  User ubuntu
  StrictHostKeyChecking=no
  UserKnownHostsFile=/dev/null
  IdentityFile ~/.ssh/sampledeployment001_rsa
Host <genesis_bastion_ip>
  User ubuntu
  StrictHostKeyChecking=no
  UserKnownHostsFile=/dev/null
  IdentityFile ~/.ssh/sampledeployment002_rsa
```

## Set environment variables

Just for convenience, let's store deploy tool arguments and other configuration settings in shell variables, for example:
```
export capideploy=../../build/capideploy.exe
export DEPLOY_ARGS="-prj capideploy_project_dreamhost.json -prj_params $HOME/capideploy_project_params_dreamhost.json"
export DEPLOY_ROOT_KEY=$HOME/.ssh/sampledeployment001_rsa
export BASTION_IP=<dreamhost_bastion_ip>
```

## Build Capillaries components

Build deploy tool to run on your dev/devops machine (this is a Windows example):

```
cd ./test/deploy
go build -o ../../build/capideploy.exe -ldflags="-s -w" ../../pkg/exe/deploy/capideploy.go
```

Build binaries for the cloud

```
cd ./test/deploy
GOOS=linux GOARCH=amd64 go build -o ../../build/linux/amd64/capidaemon -ldflags="-s -w" ../../pkg/exe/daemon/capidaemon.go
gzip -f ../../build/linux/amd64/capidaemon
GOOS=linux GOARCH=amd64 go build -o ../../build/linux/amd64/capiwebapi -ldflags="-s -w" ../../pkg/exe/webapi/capiwebapi.go
gzip -f ../../build/linux/amd64/capiwebapi
GOOS=linux GOARCH=amd64 go build -o ../../build/linux/amd64/capitoolbelt -ldflags="-s -w" ../../pkg/exe/toolbelt/capitoolbelt.go
gzip -f ../../build/linux/amd64/capitoolbelt
```

Build [Capillaries UI](../../doc/glossary.md#capillaries-ui):

```
cd ./ui
set CAPILLARIES_WEBAPI_URL=http://$BASTION_IP
npm run build
```

## Openstack networking and volumes

This step does not have to be performed often, assuming the Openstack provider does not charge for networking and volumes.

```
cd ./test/deploy
$capideploy create_security_groups $DEPLOY_ARGS
$capideploy create_networking $DEPLOY_ARGS
$capideploy create_volumes $DEPLOY_ARGS
```

## Prepare test data

This command will populate /tmp/capi_in, /tmp/capi_cfg, /tmp/capi_out

```
cd ./test/code/lookup
./1_create_quicktest_data.sh
./create_bigtest_data.sh
cd test/code/tag_and_denormalize
./1_create_quicktest_data.sh
```

Deployment projects are configured to tell deploy tool to pick up the files to upload from those locations.

## Build test environment 

```
# Create all instances in one shot
$capideploy create_instances all $DEPLOY_ARGS

# Make sure we can actually login to each instance. If an instance is missing for too long, go to the provider console/logs for details
$capideploy ping_instances all $DEPLOY_ARGS

# Install all pre-requisite software
$capideploy install_services all $DEPLOY_ARGS

# Now it's a good time to start Cassandra cluster in a separate shell session (see next section)

# Create sftp user (we assume that Openstack provider does not support multi-attach volumes, so we have to use sftp to read and write data files)
$capideploy create_instance_users bastion $DEPLOY_ARGS

# Allow these instances to connect to data via sftp
$capideploy copy_private_keys bastion,daemon01,daemon02 $DEPLOY_ARGS

# Attach volumes and make sftpuser owner
$capideploy attach_volumes bastion $DEPLOY_ARGS

# Upload all files in one shot. Make sure you have all binaries and test data built before uploading them (see above).
$capideploy upload_files up_daemon_binary,up_daemon_env_config,up_webapi_env_config,up_webapi_binary,up_ui,up_toolbelt_env_config,up_toolbelt_binary,up_all_cfg,up_lookup_bigtest_in,up_lookup_bigtest_out,up_lookup_quicktest_in,up_lookup_quicktest_out,up_tag_and_denormalize_quicktest_in,up_tag_and_denormalize_quicktest_out,up_py_calc_quicktest_in,up_py_calc_quicktest_out $DEPLOY_ARGS

# Configure all services except Cassandra (which requires extra care)
$capideploy config_services bastion,rabbitmq,prometheus,daemon01,daemon02 $DEPLOY_ARGS
```

## Starting cassandra cluster

This is probably the most fragile part of the provisioning process, as Cassandra nodes, if started simultaneously, may get into token collision situation. To avoid it, consider two approaches.

### Add nodes to Cassandra cluster one by one

The script below calls `config_service` deploy command for each Cassandra node and waits until `nodetool status` confirms that the node joined the cluster. It's worth running this script in a separate shell session right after `install_services` command is complete.

Keep in mind that `config_service` command also restarts Cassandra on each node.

```
#! /bin/bash

echo Stopping all nodes in the cluster...
$capideploy stop_services cass01,cass02,cass03,cass04,cass05 $DEPLOY_ARGS

cassNodes=('cass01:10.5.0.11' 'cass02:10.5.0.12' 'cass03:10.5.0.13' 'cass04:10.5.0.14' 'cass05:10.5.0.15')
for cassNode in ${cassNodes[@]}
do
  cassNodeParts=(${cassNode//:/ })
  cassNodeNickname=${cassNodeParts[0]}
  cassNodeIp=${cassNodeParts[1]}
  echo Configuring $cassNodeNickname $cassNodeIp...
  $capideploy config_services $cassNodeNickname $DEPLOY_ARGS
  if [ "$?" -ne "0" ]; then
    echo Cannot configure Cassandra on $cassNodeNickname
    return $?
  fi
  while :
  do
    nodetoolOutput=$(ssh -o StrictHostKeyChecking=no -i $DEPLOY_ROOT_KEY -J $BASTION_IP ubuntu@$cassNodeIp 'nodetool status' 2>&1)
    if [[ "$nodetoolOutput" == *"UJ  $cassNodeIp"* ]]; then
      echo $cassNodeNickname is joining the cluster, almost there...
    elif [[ "$nodetoolOutput" == *"InstanceNotFoundException"* ]]; then
      echo $cassNodeNickname is not started yet, getting instance not found exception
    elif [[ "$nodetoolOutput" == *"nodetool: Failed to connect"* ]]; then 
      echo $cassNodeNickname is not online, nodetool cannot connect to 7199 
    elif [[ "$nodetoolOutput" == *"Has this node finished starting up"* ]]; then 
      echo $cassNodeNickname is not online, still starting up 
    elif [[ "$nodetoolOutput" == *"UN  $cassNodeIp"* ]]; then
      echo $cassNodeNickname joined the cluster
      break
    elif [[ "$nodetoolOutput" == *"Normal/Leaving/Joining/Moving"* ]]; then
      echo $cassNodeNickname is about to start joining the cluster, nodetool functioning
    else
      echo $nodetoolOutput
    fi
    sleep 10
  done
done
ssh -o StrictHostKeyChecking=no -i $DEPLOY_ROOT_KEY -J $BASTION_IP ubuntu@10.5.0.11 'nodetool status'
```

### Speed up bootstrapping

Alternatively, you may decide not to wait for each node to join the cluster, and provide INITIAL_TOKEN setting for each Cassandra node to speed up the bootstrapping pocess. For example, for a 5-node cluster:

| Node | Project setting |
|- | - |
| cass01 | "INITIAL_TOKEN": "-8986562708977859996,-8008417256461745920,-686298450977230434,-6229782955638158791,-5133741055403220502,-4042443944711008242,-3198067297219846486,-1757799067317263246,1149224886377547375,2224980767603126914,2960896460314020555,331552953765196581,4040269267761622213,5283789273442214185,6251153516520749684,7841158069034648322" |
| cass02 | "INITIAL_TOKEN": "-8388099018021121169,-75477045641999772,-7353798371345716802,-6927511870064587412,-5577310043489909528,-4519933526560951311,-2398238626558716202,-1139818035415063168,1744527255337946625,3621591584850932654,4790546508029947033,5839670308359138121,6775583481150653541,7191221140722293421,8402901487394274566,8800224339742987357" |
| cass03 | "INITIAL_TOKEN": "-8621714953883084700,-7643824097489596036,-6496500389133919328,-5829569640212978492,-4772012955522929933,-3529574640842797781,-313735575954833661,-2702728184104684511,-2038219481927426436,-1376087975625185200,2656306692390144012,3370602156787805005,4487840607659502599,7559564332349068739,824390963435061329,9192045296353332790", |
| cass04 | "INITIAL_TOKEN": "-913058243196146801,-7826120676975670978,-7140655120705152107,-6363141672386039060,-3786009292776903012,-2550483405331700357,1446876070857747000,2440643729996635463,3830930426306277433,5037167890736080609,577971958600128955,6045411912439943902,6513368498835701612,7375392736535681080,8122029778214461444,8601562913568630961" |
| cass05 | "INITIAL_TOKEN": "-7034083495384869760,-5355525549446565015,-4645973241041940622,-4281188735635979777,-3363820969031322134,-1898009274622344841,-1257953005520124184,1984754011470536769,2808601576352082283,4264054937710562406,5561729790900676153,6644475989993177576,7467478534442374909,8262465632804368005,8996134818048160073,986807924906304352" |

Initial token values can be borrowed from a running cluster using `nodetool ring` command, or pre-calculated using some custom tool. 

While you may be tempted to configure all nodes at once with `$capideploy config_services cass01,cass02,cass03,cass04,cass05 $DEPLOY_ARGS` command, this approach does not guarantee that Cassandra nodes do not throw errors like this:
`Other bootstrapping/leaving/moving nodes detected, cannot bootstrap while cassandra.consistent.rangemovement is true`. To mitigate this issue, try starting nodes one by one with some interval:

```
$capideploy config_services cass01 $DEPLOY_ARGS
sleep 30
$capideploy config_services cass02 $DEPLOY_ARGS
sleep 30
$capideploy config_services cass03 $DEPLOY_ARGS
sleep 30
$capideploy config_services cass04 $DEPLOY_ARGS
sleep 30
$capideploy config_services cass05 $DEPLOY_ARGS
```

After that, if `nodetool status` shows than some nodes did not join the cluster, try `stop_services` and `start_services` commands for each node in question, one by one. Or, start troubleshooting nodes by examining their `/var/log/cassandra/debug.log` files.

## Monitoring test environment

RabbitMQ console (use RabbitMQ username/password from the project parameter file): `http://$BASTION_IP:15672`

Resource usage:

| Metric | Prometheus screen |
|- | - |
| CPU usage % | `http://$BASTION_IP:9090/graph?g0.expr=(1%20-%20avg(irate(node_cpu_seconds_total%7Bmode%3D%22idle%22%7D%5B10m%5D))%20by%20(instance))%20*%20100&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=15m` |
| RAM usage % | `http://$BASTION_IP:9090/graph?g0.expr=100%20*%20(1%20-%20((avg_over_time(node_memory_MemFree_bytes%5B10m%5D)%20%2B%20avg_over_time(node_memory_Cached_bytes%5B10m%5D)%20%2B%20avg_over_time(node_memory_Buffers_bytes%5B10m%5D))%20%2F%20avg_over_time(node_memory_MemTotal_bytes%5B10m%5D)))&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=15m` |
| Disk usage % | `http://$BASTION_IP:9090/graph?g0.expr=100%20-%20((node_filesystem_avail_bytes%7Bmountpoint%3D%22%2F%22%2Cfstype!%3D%22rootfs%22%7D%20*%20100)%2Fnode_filesystem_size_bytes%7Bmountpoint%3D%22%2F%22%2Cfstype!%3D%22rootfs%22%7D)&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=15m` |

Consolidated [Daemon](../../doc/glossary.md#daemon) log from all Daemon instances:

```
ssh -o StrictHostKeyChecking=no -i $DEPLOY_ROOT_KEY ubuntu@$BASTION_IP
less /var/log/capidaemon/capidaemon.log
```

## Run test processes

Start runs either using [Webapi](../../doc/glossary.md#webapi) at `http://$BASTION_IP` or using [Toolbelt](../../doc/glossary.md#toolbelt) via SSH.

### lookup_quicktest

| Field | Value |
|- | - |
| Keyspace | lookup_quicktest |
| Script URI | sftp://sftpuser@10.5.0.10/mnt/capi_cfg/lookup_quicktest/script.json |
| Script parameters URI | sftp://sftpuser@10.5.0.10/mnt/capi_cfg/lookup_quicktest/script_params_one_run.json |
| Start nodes |	read_orders,read_order_items |

or

```
ssh -o StrictHostKeyChecking=no -i $DEPLOY_ROOT_KEY ubuntu@$BASTION_IP '~/bin/capitoolbelt start_run -script_file=sftp://sftpuser@10.5.0.10/mnt/capi_cfg/lookup_quicktest/script.json -params_file=sftp://sftpuser@10.5.0.10/mnt/capi_cfg/lookup_quicktest/script_params_one_run.json -keyspace=lookup_quicktest -start_nodes=read_orders,read_order_items'
```

### lookup_bigtest

| Field | Value |
|- | - |
| Keyspace | lookup_bigtest |
| Script URI | sftp://sftpuser@10.5.0.10/mnt/capi_cfg/lookup_bigtest/script.json |
| Script parameters URI | sftp://sftpuser@10.5.0.10/mnt/capi_cfg/lookup_bigtest/script_params_one_run.json |
| Start nodes |	read_orders,read_order_items |

or

```
ssh -o StrictHostKeyChecking=no -i $DEPLOY_ROOT_KEY ubuntu@$BASTION_IP '~/bin/capitoolbelt start_run -script_file=sftp://sftpuser@10.5.0.10/mnt/capi_cfg/lookup_bigtest/script.json -params_file=sftp://sftpuser@10.5.0.10/mnt/capi_cfg/lookup_bigtest/script_params_one_run.json -keyspace=lookup_bigtest -start_nodes=read_orders,read_order_items'
```

### py_calc_quicktest

| Field | Value |
|- | - |
| Keyspace | py_calc_bigtest |
| Script URI | sftp://sftpuser@10.5.0.10/mnt/capi_cfg/py_calc_bigtest/script.json |
| Script parameters URI | sftp://sftpuser@10.5.0.10/mnt/capi_cfg/py_calc_bigtest/script_params.json |
| Start nodes |	read_order_items |

or

```
ssh -o StrictHostKeyChecking=no -i $DEPLOY_ROOT_KEY ubuntu@$BASTION_IP '~/bin/capitoolbelt start_run -script_file=sftp://sftpuser@10.5.0.10/mnt/capi_cfg/py_calc_quicktest/script.json -params_file=sftp://sftpuser@10.5.0.10/mnt/capi_cfg/py_calc_quicktest/script_params.json -keyspace=py_calc_quicktest -start_nodes=read_order_items'
```

### tag_and_denormalize_quicktest

| Field | Value |
|- | - |
| Keyspace | tag_and_denormalize_quicktest |
| Script URI | sftp://sftpuser@10.5.0.10/mnt/capi_cfg/tag_and_denormalize_quicktest/script.json |
| Script parameters URI | sftp://sftpuser@10.5.0.10/mnt/capi_cfg/tag_and_denormalize_quicktest/script_params_one_run.json |
| Start nodes |	read_tags,read_products |

or

```
ssh -o StrictHostKeyChecking=no -i $DEPLOY_ROOT_KEY ubuntu@$BASTION_IP '~/bin/capitoolbelt start_run -script_file=sftp://sftpuser@10.5.0.10/mnt/capi_cfg/tag_and_denormalize_quicktest/script.json -params_file=sftp://sftpuser@10.5.0.10/mnt/capi_cfg/tag_and_denormalize_quicktest/script_params_one_run.json -keyspace=tag_and_denormalize_quicktest -start_nodes=read_tags,read_products'
```

## Results

Download all results from capi_out:

```
$capideploy download_files down_capi_out $DEPLOY_ARGS
```

Download consolidated Daemon log:

```
$capideploy download_files down_capi_logs $DEPLOY_ARGS
```

## Clear test environment

```
# Shut down instances to save money
$capideploy delete_instances all $DEPLOY_ARGS

# Delete everything (does not have to be performed often)
$capideploy delete_volumes $DEPLOY_ARGS
$capideploy delete_networking $DEPLOY_ARGS
$capideploy delete_security_groups $DEPLOY_ARGS
```

## Q&A

### Wordy configuration

Q. Why does project file contain repetitive configuration for each instance? Why not using templates?

A. When deploying to the cloud, deploy tool writes the current status of each cloud resource (instance, volume etc) to the project file. So, project file is not just a recipe, it's also a snapshot of the cloud deployment status. It can go out of sync, in which case a user should be able to easily tweak it (usually, make the id of a deleted resource empty). If you need a more concise configuration structure, consider developing a templating mechanism that would take a recipe and produce a Capillaries deploy project JSON file.

### Openstack environment variables

Q. The list of `OS_*` variables changes from one Openstack provider to another. Why?

A. The choice of Openstack variables required for authentication is up to the provider. Some providers may require more variables than other.

### Service: install vs. config

Q. Why having separate sets of commands for software installation and configuration?

A. Installing all Capillaries pre-requisites is a one-time job and can take a while. Configuring services is a job that:
- usually takes less less time to execute 
- can be executed multiple times per instance (like re-configuring Daemon thread parameters)
- may be required to be executed in some specific order (like adding nodes to Cassandra cluster)

### SFTP vs. multi-attach volumes

Q. Why these sample deployment projects use SFTP to read/write test data? Is there a more straightforward way?

A. Volume capabilities differ from one provider to another. Some providers simply do not support multi-attach volumes, while SFTP is something that can be configured in any cloud. If your provider supports multi-attach volumes, feel free to change all file URIs used in your project from SFTP scheme to, say, `/mnt/capi_*`, so Webapi/Toolbelt/Daemon read/write directly from/to your volume.

### Non-Openstack clouds

Q. Does Deploy tool work with clouds that do not support Openstack? AWS,Azure,GCP?

A. At the moment, no.

### Why should I use another custom deploy tool?

Q. I am familiar with widely used infrastructure provisioning tools (Ansible, Saltstack etc). Can I use them instead of Capillaries Deploy tool?

A. Absolutely. Capillaries Deploy tool was created to serve only one goal: to demonstrate that production-scale Capillaries deployment can be provisioned very quickly (within a few minutes) without using complex third-party software.
