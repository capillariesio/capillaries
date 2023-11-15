#! /bin/bash

# This script configures and starts Cassandra cluster configured in $2.
# It is not called by capideploy, it should be executed manually. See test/deploy/README.md for details.

# Troubleshooting. Sometimes, a node cannot start because of schema version mismatch. Then:
# on the failing node:
# sudo systemctl stop cassandra
# sudo rm -fR /var/lib/cassandra/data/*
# sudo rm -fR /var/lib/cassandra/commitlog/*
# on a working node - remove failing node from cluster:
# nodetool removenode <failing host id taken from nodetool status>
# on the failing node:
# sudo systemctl start cassandra
# and wait until the node actually starts (1-2 min)

if ! command -v jq &> /dev/null
then
    echo "jq command could not be found, please install it before running this script, e.g. 'apt-get install -y jq'"
    exit 1
fi
if [ "$1" != "parallel" ] && [ "$1" != "sequential" ]; then
  echo Error, missing parallel/sequential parameter
  exit 1
fi
if [ "$2" = "" ]; then
  echo Error, missing deployment project file parameter, e.g. sampledeployment002.json
  exit 1
fi
if [ "$capideploy" = "" ]; then
  echo Error, missing 'capideploy' env variable pointing to the capideploy command
  exit 1
fi
if [ "$CAPIDEPLOY_SSH_USER" = "" ]; then
  echo Error, missing CAPIDEPLOY_SSH_USER env variable with the name of the SSH user to use
  exit 1
fi
sshKeyFile=$(jq -r .ssh_config.private_key_path $2)
if [ "$sshKeyFile" = "" ]; then
  echo Error, missing ssh_config.private_key_path in $2
  exit 1
fi
externalIpAddress=$(jq -r .ssh_config.external_ip_address $2)
if [ "$externalIpAddress" = "" ]; then
  echo Error, missing ssh_config.external_ip_address in $2
  exit 1
fi

# Find string containing all cass ips: 10.5.0.11,10.5.0.12,10.5.0.13,10.5.0.14,10.5.0.15,10.5.0.16,10.5.0.17,10.5.0.18
cassIpString=$(jq -r .instances.bastion.service.env.CASSANDRA_HOSTS $2 | sed -e 's/[^0-9.,]//g')

# Make it an array of ips
cassIpArray=(${cassIpString//,/ })

if [ "${#cassIpArray[@]}" = "0" ]; then
  echo Error, the number of Cassandra nodes is zero
  exit 1
fi

# Build node array
cassNodes=() # ('cass001:10.5.0.11' 'cass002:10.5.0.12' 'cass003:10.5.0.13' 'cass004:10.5.0.14' 'cass005:10.5.0.15' 'cass006:10.5.0.16' 'cass007:10.5.0.17' 'cass008:10.5.0.18')
cassNames=() # (cass001 cass002 cass003 cass004 cass005 cass006 cass007 cass008)
for i in "${!cassIpArray[@]}"; do
  cassNames[$i]=cass$(printf "%03d" $((i+1)) );
  cassNodes[$i]=cass$(printf "%03d" $((i+1)) )":"${cassIpArray[$i]};
done

# All cassXXX in one comma-separated string: cass001,cass002,cass003,cass004,cass005,cass006,cass007,cass008
cassNamesString=$(IFS=, ; echo "${cassNames[*]}")

echo Stopping all nodes in the cluster...
$capideploy stop_services $cassNamesString -prj="$2"

if [ "$1" = "sequential" ]; then
  for cassNode in ${cassNodes[@]}
  do
    cassNodeParts=(${cassNode//:/ })
    cassNodeNickname=${cassNodeParts[0]}
    cassNodeIp=${cassNodeParts[1]}
    echo Configuring $cassNodeNickname $cassNodeIp...
    $capideploy config_services $cassNodeNickname -prj="$2"
    if [ "$?" -ne "0" ]; then
      echo Cannot configure Cassandra on $cassNodeNickname
      return $?
    fi
    while :
    do
      nodetoolOutput=$(ssh -o StrictHostKeyChecking=no -i $sshKeyFile -J $externalIpAddress $CAPIDEPLOY_SSH_USER@$cassNodeIp 'nodetool status' 2>&1)
      if [[ "$nodetoolOutput" == *"UJ  $cassNodeIp"* ]]; then
        echo $cassNodeNickname is joining UJ the cluster, almost there, starting next node...
        # Should be ok to consider this node completed, comment this out if we really need to wait for UN
        break
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
        echo $cassNodeNickname is about to start joining the cluster, nodetool functioning, but no UN/UJ yet...
      else
        echo $nodetoolOutput
      fi
      sleep 5
    done
  done
  ssh -o StrictHostKeyChecking=no -i $sshKeyFile -J $externalIpAddress $CAPIDEPLOY_SSH_USER@${cassIpArray[0]} 'nodetool describecluster;nodetool status'
else
  for cassNode in ${cassNodes[@]}
  do
    cassNodeParts=(${cassNode//:/ })
    cassNodeNickname=${cassNodeParts[0]}
    cassNodeIp=${cassNodeParts[1]}
    echo Configuring $cassNodeNickname $cassNodeIp...
    $capideploy config_services $cassNodeNickname -prj="$2"
    if [ "$?" -ne "0" ]; then
      echo Cannot configure Cassandra on $cassNodeNickname
      return $?
    fi
  done
  sleep 10
  ssh -o StrictHostKeyChecking=no -i $sshKeyFile -J $externalIpAddress $CAPIDEPLOY_SSH_USER@${cassIpArray[0]} 'nodetool describecluster;nodetool status'
fi