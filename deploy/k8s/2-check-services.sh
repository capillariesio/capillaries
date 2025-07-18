#!/bin/bash

RED="\e[31m"
GREEN="\e[32m"
ENDCOLOR="\e[0m"

# Some sample deployment ready checks
while true
do
  kubectl get pods
  cass=$(kubectl get pods|grep -E 'cassandra-[0-9][ ]+1/1[ ]+Running'|wc -l)
  daemon=$(kubectl get pods|grep -E 'daemon-[a-z0-9-]+[ ]+1/1[ ]+Running'|wc -l)
  rabbitmq=$(kubectl get pods|grep -E 'rabbitmq-[0-9][ ]+1/1[ ]+Running'|wc -l)
  ui=$(kubectl get pods|grep -E 'ui-[a-z0-9-]+[ ]+1/1[ ]+Running'|wc -l)
  webapi=$(kubectl get pods|grep -E 'webapi-[a-z0-9-]+[ ]+1/1[ ]+Running'|wc -l)
  if [ "$cass" == "2" ] && [ "$daemon" == "2" ] && [ "$rabbitmq" == "1" ] && [ "$ui" == "1" ] && [ "$webapi" == "1" ]; then
    echo -e "Pods ${GREEN}OK${ENDCOLOR}"
    break;
  fi
  echo -e "Pods ${RED}NOT READY${ENDCOLOR}"
  sleep 5
done

while true
do
  kubectl exec -it 'cassandra-0' -- env JVM_OPTS= nodetool status
  if [ "$(kubectl exec -it 'cassandra-0' -- env JVM_OPTS= nodetool status|grep 'UN  '|wc -l)" == "2" ]; then
    echo -e "Cassandra cluster ${GREEN}OK${ENDCOLOR}"
    break;
  fi
  echo -e "Cassandra cluster ${RED}NOT READY${ENDCOLOR}"
  sleep 5
done

while true
do
  metricLines=$(kubectl exec -it 'cassandra-1' -- /bin/sh -c 'curl localhost:7070/metric'|wc -l)
  if [ "$metricLines" -gt "100" ]; then
    echo -e "JMX exporter ${GREEN}OK${ENDCOLOR}, $metricLines lines"
    break;
  fi
  echo -e "JMX exporter ${RED}NOT READY${ENDCOLOR}, $metricLines lines"
  sleep 5
done

while true
do
  webapiResponse=$(kubectl exec -it 'cassandra-1' -- curl webapi.default.svc.cluster.local:6543/ks)
  if [[ $webapiResponse == *'"error":{"msg":""}'* ]]; then
    echo -e "Webapi ${GREEN}OK${ENDCOLOR}"
    break;
  fi
  echo -e "Webapi ${RED}NOT READY${ENDCOLOR}: $webapiResponse"
  sleep 5
done

webapiPod=$(kubectl get pods|grep webapi|awk '{print $1}')

while true
do
  dbConn=$(kubectl exec -it $webapiPod -- capitoolbelt check_db_connectivity | grep "OK:" | wc -l)
  if [ "$dbConn" == "1" ]; then
    echo -e "DB connectivity ${GREEN}OK${ENDCOLOR}"
    break;
  fi
  echo -e "DB connectivity ${RED}NOT READY${ENDCOLOR}"
  sleep 5
done

while true
do
  queueConn=$(kubectl exec -it $webapiPod -- capitoolbelt check_queue_connectivity | grep "OK:" | wc -l)
  if [ "$queueConn" == "1" ]; then
    echo -e "Queue connectivity ${GREEN}OK${ENDCOLOR}"
    break;
  fi
  echo -e "Queue connectivity ${RED}NOT READY${ENDCOLOR}"
  sleep 5
done

echo Ready to run ingress and test
