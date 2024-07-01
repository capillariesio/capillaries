#!/bin/bash

# Verify Capillaries are deployed somewhere at $BASTION_IP with SSH key $CAPIDEPLOY_SSH_PRIVATE_KEY_PATH
check_cloud_deployment()
{
    if [ "$CAPIDEPLOY_SSH_PRIVATE_KEY_PATH" = "" ]; then
        echo Error, missing: export CAPIDEPLOY_SSH_PRIVATE_KEY_PATH=~/.ssh/mydeployment005_rsa
        echo This is the SSH private key used to access hosts in your Capilaries cloud deployment
        echo See capillaries-deploy repo for details
        exit 1
    fi
    if [ "$BASTION_IP" = "" ]; then
        echo Error, missing: export BASTION_IP=1.2.3.4
        echo This is the ip address of the bastion host in your Capilaries cloud deployment
        echo See capillaries-deploy repo for details
        exit 1
    fi
    if [ "$CAPIDEPLOY_EXTERNAL_WEBAPI_PORT" = "" ]; then
        echo Error, missing: export CAPIDEPLOY_EXTERNAL_WEBAPI_PORT=6544
        echo "This is the external (proxied) port of the webapi in your Capilaries cloud deployment"
        echo See capillaries-deploy repo for details
        exit 1
    fi
}

# Verify s3 credentials and bucket are specified
check_s3()
{
    if [ "$CAPILLARIES_AWS_TESTBUCKET" = "" ]; then
        echo Error, missing: export CAPILLARIES_AWS_TESTBUCKET=capillaries-testbucket
        echo This is the name of the bucket the user creates to test S3 Capillaries scenarios.
        echo See s3.md for details on how to set up bucket permissions.
        exit 1
    fi

    if [ "$AWS_ACCESS_KEY_ID" == "" ]; then
        echo Error, please specify export AWS_ACCESS_KEY_ID=...
        exit 1
    fi
    if [ "$AWS_SECRET_ACCESS_KEY" == "" ]; then
        echo Error, please specify export AWS_SECRET_ACCESS_KEY=...
        exit 1
    fi

    if [ "$AWS_DEFAULT_REGION" == "" ]; then
        echo Error, please specify export AWS_DEFAULT_REGION=...
        exit 1
    fi
}


# Makes toolbelt call get_run_history in a loop until run status is as requested
wait()
{
    local keyspace=$1
    local runIdToCheck=$2
    local statusToCheck=$3
    local outDir=$4
    while true
    do
        go run capitoolbelt.go get_run_history -keyspace=$keyspace > $outDir/runs.csv
        while IFS="," read -r ts run_id status comment
        do
            if [ $run_id -eq $runIdToCheck ]; then
                if [ "$statusToCheck" -eq "1" ]; then # Wait for start
                    if [ "$status" -eq "1" ]; then  
                        echo Run started
                        return
                    fi  
                else # Wait for completion or stop signal
                    if [ "$status" -eq "2" ]; then  
                        echo Finished
                        return
                    else
                        if [ "$status" -eq "3" ]; then
                            echo Stopped
                            return
                        fi
                    fi  
                fi
            fi
        done < <(tail -n +2 $outDir/runs.csv)
        echo "Waiting keyspace $keyspace run $runIdToCheck status $statusToCheck..."
        sleep 1
    done
}

one_daemon_run()
{
    local keyspace=$1
    local scriptFile=$2
    local paramsFile=$3
    local outDir=$4
    local startNodes=$5

    echo Starting $scriptFile with $paramsFile in $keyspace...

    SECONDS=0

    # A hack to support *_quicktest additional dir level
    if [ -d "../../../../pkg/exe/toolbelt" ]; then
        pushd ../../../../pkg/exe/toolbelt
    else
        pushd ../../../pkg/exe/toolbelt
    fi

    go run capitoolbelt.go drop_keyspace -keyspace=$keyspace
    if [ "$?" = "0" ]; then
      go run capitoolbelt.go start_run -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -start_nodes=$startNodes
      if [ "$?" = "0" ]; then
        echo "Waiting for run to start..."
        wait $keyspace 1 1 $outDir
        echo "Waiting for run to finish, make sure pkg/exe/daemon is running..."
        wait $keyspace 1 2 $outDir
        go run capitoolbelt.go get_node_history -keyspace=$keyspace -run_ids=1
      fi
    fi
    popd
    duration=$SECONDS
    echo "$(($duration / 60))m $(($duration % 60))s elapsed."    
}

one_daemon_run_no_params()
{
    local keyspace=$1
    local scriptFile=$2
    local outDir=$3
    local startNodes=$4

    echo Starting $scriptFile in $keyspace...

    SECONDS=0

    # A hack to support *_quicktest additional dir level
    if [ -d "../../../../pkg/exe/toolbelt" ]; then
        pushd ../../../../pkg/exe/toolbelt
    else
        pushd ../../../pkg/exe/toolbelt
    fi

    go run capitoolbelt.go drop_keyspace -keyspace=$keyspace
    if [ "$?" = "0" ]; then
      go run capitoolbelt.go start_run -script_file=$scriptFile -keyspace=$keyspace -start_nodes=$startNodes
      if [ "$?" = "0" ]; then
        echo "Waiting for run to start..."
        wait $keyspace 1 1 $outDir
        echo "Waiting for run to finish, make sure pkg/exe/daemon is running..."
        wait $keyspace 1 2 $outDir
        go run capitoolbelt.go get_node_history -keyspace=$keyspace -run_ids=1
      fi  
    fi
    popd
    duration=$SECONDS
    echo "$(($duration / 60))m $(($duration % 60))s elapsed."    
}

two_daemon_runs()
{
    local keyspace=$1
    local scriptFile=$2
    local paramsFile=$3
    local outDir=$4
    local startNodesOne=$5
    local startNodesTwo=$6

    SECONDS=0

    # A hack to support *_quicktest additional dir level
    if [ -d "../../../../pkg/exe/toolbelt" ]; then
        pushd ../../../../pkg/exe/toolbelt
    else
        pushd ../../../pkg/exe/toolbelt
    fi

    go run capitoolbelt.go drop_keyspace -keyspace=$keyspace
    if [ "$?" = "0" ]; then
        # Operator starts run 1

        go run capitoolbelt.go start_run -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -start_nodes=$startNodesOne
        if [ "$?" = "0" ]; then
            echo "Waiting for run to start..."
            wait $keyspace 1 1 $outDir
            echo "Waiting for run to finish, make sure pkg/exe/daemon is running..."
            wait $keyspace 1 2 $outDir
            go run capitoolbelt.go get_node_history -keyspace=$keyspace -run_ids=1

            # Operator approves intermediate results and starts run 2

            go run capitoolbelt.go start_run -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -start_nodes=$startNodesTwo
            echo "Waiting for run to start..."
            wait $keyspace 2 1 $outDir
            echo "Waiting for run to finish, make sure pkg/exe/daemon is running..."
            wait $keyspace 2 2 $outDir
            go run capitoolbelt.go get_node_history -keyspace=$keyspace -run_ids=1,2
        fi
    fi    
    popd
    duration=$SECONDS
    echo "$(($duration / 60))m $(($duration % 60))s elapsed."
}

wait_run_webapi()
{
    local webapiUrl=$1
    local keyspace=$2
    local runIdToCheck=$3
    while true
    do
      runNodeHistoryCmd="curl -s -X GET ""$webapiUrl/ks/$keyspace/run/$runIdToCheck/node_history"""
      runNodeHistory=$($runNodeHistoryCmd)
      string='My long string'
      if [[ $runNodeHistory == *"\"final_status\":1"* ]]; then
        echo "Run $runIdToCheck running, waiting..."
      elif [[ $runNodeHistory == *"\"final_status\":2"* ]]; then
        echo "Run $runIdToCheck completed"
        return
      elif [[ $runNodeHistory == *"\"final_status\":3"* ]]; then
        echo "Run $runIdToCheck was stopped"
        return
      fi
      sleep 2
    done
}

one_daemon_run_webapi()
{
    local webapiUrl=$1
    local keyspace=$2
    local scriptFile=$3
    local paramsFile=$4
    local startNodes=$5

    SECONDS=0
    echo Deleting keyspace $keyspace at $webapiUrl ...
    curl -s -w "\n" -H "Content-Type: application/json" -X DELETE $webapiUrl"/ks/"$keyspace
    if [ "$?" != "0" ]; then
      exit 1
    fi

    echo Starting a run in $keyspace at $webapiUrl ...
    curl -s -w "\n" -d '{"script_uri":"'$scriptFile'", "script_params_uri":"'$paramsFile'", "start_nodes":"'$startNodes'"}' -H "Content-Type: application/json" -X POST $webapiUrl"/ks/$keyspace/run"
    if [ "$?" != "0" ]; then
      exit 1
    fi

    wait_run_webapi $webapiUrl $keyspace 1

    duration=$SECONDS
    echo "$(($duration / 60))m $(($duration % 60))s elapsed."
}
