#!/bin/bash

# Assumptions:
# - this script is run from test/py_calc
# - python interpreter is available by name 'python'

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

keyspace="test_py_calc"
scriptDir="../../../test/py_calc"
outDir="../../../test/py_calc/data/out"
scriptFile=$scriptDir/script.json
paramsFile=$scriptDir/script_params.json

# HTTP(S) script URIs are supported. They slow down things a lot.
#scriptFile=https://github.com/kleineshertz/capillaries/blob/main/test/py_calc/script.json?raw=1
#paramsFile=https://github.com/kleineshertz/capillaries/blob/main/test/py_calc/script_params.json?raw=1

SECONDS=0
[ ! -d "./data/out" ] && mkdir ./data/out
pushd ../../pkg/exe/toolbelt
go run toolbelt.go drop_keyspace -keyspace=$keyspace
go run toolbelt.go start_run -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -start_nodes=read_order_items
echo "Waiting for run to start..."
wait $keyspace 1 1 $outDir
echo "Waiting for run to finish, make sure pkg/exe/daemon is running..."
wait $keyspace 1 2 $outDir
go run toolbelt.go get_node_history -keyspace=$keyspace -run_ids=1
popd
duration=$SECONDS
echo "$(($duration / 60))m $(($duration % 60))s elapsed."
