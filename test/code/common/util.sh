#!/bin/bash

wait()
{
    local keyspace=$1
    local runIdToCheck=$2
    local statusToCheck=$3
    local outDir=$4
    while true
    do
        go run toolbelt.go get_run_history -keyspace=$keyspace > $outDir/runs.csv
        while IFS="," read -r ts run_id status comment
        do
            if [ "$run_id" -eq "$runIdToCheck" ]; then
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

    SECONDS=0
    pushd ../../../pkg/exe/toolbelt

    go run toolbelt.go drop_keyspace -keyspace=$keyspace
    
    go run toolbelt.go start_run -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -start_nodes=$startNodes
    echo "Waiting for run to start..."
    wait $keyspace 1 1 $outDir
    echo "Waiting for run to finish, make sure pkg/exe/daemon is running..."
    wait $keyspace 1 2 $outDir
    go run toolbelt.go get_node_history -keyspace=$keyspace -run_ids=1

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
    pushd ../../../pkg/exe/toolbelt

    go run toolbelt.go drop_keyspace -keyspace=$keyspace

    # Operator starts run 1

    go run toolbelt.go start_run -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -start_nodes=$startNodesOne
    echo "Waiting for run to start..."
    wait $keyspace 1 1 $outDir
    echo "Waiting for run to finish, make sure pkg/exe/daemon is running..."
    wait $keyspace 1 2 $outDir
    go run toolbelt.go get_node_history -keyspace=$keyspace -run_ids=1

    # Operator approves intermediate results and starts run 2

    go run toolbelt.go start_run -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -start_nodes=$startNodesTwo
    echo "Waiting for run to start..."
    wait $keyspace 2 1 $outDir
    echo "Waiting for run to finish, make sure pkg/exe/daemon is running..."
    wait $keyspace 2 2 $outDir
    go run toolbelt.go get_node_history -keyspace=$keyspace -run_ids=1,2
    
    popd
    duration=$SECONDS
    echo "$(($duration / 60))m $(($duration % 60))s elapsed."
}


two_daemon_runs_webapi()
{
    local keyspace=$1
    local scriptFile=$2
    local paramsFile=$3
    local outDir=$4
    local startNodesOne=$5
    local startNodesTwo=$6

    SECONDS=0
    pushd ../../../pkg/exe/toolbelt

    curl -H "Content-Type: application/json" -X DELETE "http://localhost:6543/ks/$keyspace"

    # Operator starts run 1

    curl -d '{"script_uri":"'"$scriptFile"'", "script_params_uri":"'"$paramsFile"'", "start_nodes":"read_orders,read_order_items"}' -H "Content-Type: application/json" -X POST "http://localhost:6543/ks/$keyspace/run"

    echo "Waiting for run to start..."
    wait $keyspace 1 1 $outDir
    echo "Waiting for run to finish, make sure pkg/exe/daemon is running..."
    wait $keyspace 1 2 $outDir

    # Operator approves intermediate results and starts run 2

    curl -d '{"script_uri":"'"$scriptFile"'", "script_params_uri":"'"$paramsFile"'", "start_nodes":"order_item_date_inner,order_item_date_left_outer,order_date_value_grouped_inner,order_date_value_grouped_left_outer"}' -H "Content-Type: application/json" -X POST "http://localhost:6543/ks/$keyspace/run"

    echo "Waiting for run to start..."
    wait $keyspace 2 1 $outDir
    echo "Waiting for run to finish, make sure pkg/exe/daemon is running..."
    wait $keyspace 2 2 $outDir

    echo "Run 2 finished"
    curl "http://localhost:6543/ks/$keyspace"
    echo "Done"

    popd
    duration=$SECONDS
    echo "$(($duration / 60))m $(($duration % 60))s elapsed."
}
