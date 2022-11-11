<script>
    export let params; 
    import { onDestroy, onMount } from "svelte";
	import Common, { handleResponse } from './Common.svelte';
	let common;
    import dayjs from "dayjs";

    let webapiData = {run_props:{}, batch_history: []};

    function setWebapiData(dataFromJson) {
		webapiData = ( !!dataFromJson ? dataFromJson : {run_props:{}, batch_history: []});

        // Calculate elapsed times for each batch
        let batchStartMap = {};
        for (let i=0; i < webapiData.batch_history.length; i++) {
            let e = webapiData.batch_history[i];
            if (e.status === 1) {
                batchStartMap[e.batch_idx] = dayjs(e.ts).valueOf();
            }
        }

        let batchEndMap = {};
        for (let i=0; i < webapiData.batch_history.length; i++) {
            let e = webapiData.batch_history[i];
            if (e.status > 1 && !(e.batch_idx in batchEndMap)) {
                batchEndMap[e.batch_idx] = dayjs(e.ts).valueOf();
            }
        }

        for (let i=0; i < webapiData.batch_history.length; i++) {
            let e = webapiData.batch_history[i];
            if (e.batch_idx in batchStartMap) {
                if (e.batch_idx in batchEndMap) {
                    webapiData.batch_history[i].elapsed = (batchEndMap[e.batch_idx] - batchStartMap[e.batch_idx])/1000;
                } else {
                    webapiData.batch_history[i].elapsed = (Date.now() - batchStartMap[e.batch_idx])/1000;
                }
            }
        }
	}

	var timer;

	function fetchData() {
		fetch("http://localhost:6543/ks/" + params.ks_name + "/run/" + params.run_id + "/node/" + params.node_name +"/batch_history")
      		.then(response => response.json())
      		.then(responseJson => { handleResponse(responseJson, setWebapiData);})
      		.catch(error => {console.log(error);});
	}

	onMount(async () => {
    	fetchData();
		timer = setInterval(fetchData, 500);
    });
	onDestroy(async () => {
    	clearInterval(timer);
    });
</script>

<style>
	img { width: 20px;	}
    th { white-space: nowrap;}
</style>

<h1>Batch history: keyspace {params.ks_name}, run {params.run_id}, node {params.node_name}</h1>
<table>
	<tbody>
        <tr>
            <td style="white-space: nowrap;">Start nodes:</td>
            <td>
                {#if webapiData.run_props.start_nodes}
                    {webapiData.run_props.start_nodes}
                {/if}
            </td>
        </tr>
        <tr>
            <td style="white-space: nowrap;">Affected nodes:</td>
            <td>
                {#if webapiData.run_props.affected_nodes}
                    {webapiData.run_props.affected_nodes}
                {/if}
            </td>
        </tr>
        <tr>
            <td style="white-space: nowrap;">Script URI:</td>
            <td>
                {#if webapiData.run_props.script_uri}
                    {webapiData.run_props.script_uri}
                {/if}
            </td>
        </tr>

        <tr>
            <td style="white-space: nowrap;">Script parameters URI:</td>
            <td>
                {#if webapiData.run_props.script_params_uri}
                    {webapiData.run_props.script_params_uri}
                {/if}
            </td>
        </tr>
    </tbody>
</table>

<Common bind:this={common} />

<table>
    <thead>
        <th>Timestamp</th>
        <th>Batch</th>
        <th>Status</th>
        <th>Elapsed</th>
        <th>First token</th>
        <th>Last token</th>
        <th>Host/Instance</th>
        <th>Thread</th>
        <th>Comment</th>
    </thead>
	<tbody>
        {#each webapiData.batch_history as e}
        <tr>
            <td style="white-space: nowrap;">{dayjs(e.ts).format("MMM D, YYYY HH:mm:ss.SSS Z")}</td>
            <td>{e.batch_idx} / {e.batches_total}</td>
            <td><img src={common.nodeStatusToIcon(e.status)} title={common.nodeStatusToText(e.status)} alt=""/></td>
            <td>{#if e.elapsed > 0} {e.elapsed} {/if}</td>
            <td>{e.first_token}</td>
            <td>{e.last_token}</td>
            <td>{e.instance}</td>
            <td>{e.thread}</td>
            <td>{e.comment}</td>
        </tr>
        {/each}
    </tbody>
</table>

