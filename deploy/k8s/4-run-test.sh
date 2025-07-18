#!/bin/bash

if [ "$CAPILLARIES_AWS_TESTBUCKET" = "" ]; then
  echo export CAPILLARIES_AWS_TESTBUCKET=capillaries-testbucket is not set, and this test uses data and configuration from S3
  exit 1
fi

# We know that our Webapi pod contains capitoolbelt (see Webapi dockerfile), so we can use capitoolbelt to start a run
webapiPod=$(kubectl get pods|grep webapi|awk '{print $1}')
kubectl exec -it $webapiPod -- /usr/local/bin/capitoolbelt start_run -script_file=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/lookup_quicktest/script.json -params_file=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/lookup_quicktest/script_params_one_run_s3.json -keyspace=lookup_quicktest_s3 -start_nodes=read_orders,read_order_items

# Watch it running and see the results
echo ""
echo 'Capillaries UI: http://localhost:8080'
echo 'Prometheus: http://localhost:9090/graph?g0.expr=sum(irate(cassandra_clientrequest_localrequests_count%7Bclientrequest%3D%22Write%22%7D%5B1m%5D))&g0.tab=0&g0.display_mode=lines&g0.show_exemplars=1&g0.range_input=5m&g1.expr=sum(irate(cassandra_clientrequest_localrequests_count%7Bclientrequest%3D%22Read%22%7D%5B1m%5D))&g1.tab=0&g1.display_mode=lines&g1.show_exemplars=0&g1.range_input=5m'
echo ""
echo When complete, check results:
echo pushd ../code/lookup/quicktest_s3
echo ./3_compare_results_s3.sh