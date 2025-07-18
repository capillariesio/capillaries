#!/bin/bash

daemons_or_cassandras=$1
number_of_instances=$2

if [[ "$daemons_or_cassandras" != "-d" && "$daemons_or_cassandras" != "-c" || \
  "$number_of_instances" != "4" && "$number_of_instances" != "8" && "$number_of_instances" != "16" && "$number_of_instances" != "32" && "$number_of_instances" != "64" ]]; then
  echo $(basename "$0") requires 2 parameters:  '-d|-c' '4|8|16|32|64'
  exit 1
fi

if [ "$BASTION_IP" = "" ]; then
  echo Error, missing: export BASTION_IP=1.2.3.4
  echo This is the ip address of the bastion host in your Capilaries cloud deployment
  exit 1
fi

if [[ "$daemons_or_cassandras" == "-d" ]]; then
  first_ip=101
  last_ip=$((100 + $number_of_instances))
else
  first_ip=11
  last_ip=$((10 + $number_of_instances))
fi

cat /dev/null > ping.txt
cat /dev/null > ping_err.txt
for i in $(seq -f "%g" $first_ip $last_ip)
do
  # echo "10.5.0.$i"
  ssh -o StrictHostKeyChecking=no -i ~/.ssh/sampledeployment005_rsa -o ConnectTimeout=5 -J $BASTION_IP ubuntu@"10.5.0.$i" 'pwd' >> ping.txt 2>>ping_err.txt & 
done
wait