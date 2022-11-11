<script>
    export let params; 
    import { onDestroy, onMount } from "svelte";
	import Common, { handleResponse } from './Common.svelte';
	let common;
    import dayjs from "dayjs";

    let webapiData = {run_props:{}, batch_history: []};

    function setWebapiData(dataFromJson) {
		webapiData = ( !!dataFromJson ? dataFromJson : {run_props:{}, batch_history: []});
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

<!-- {"ts":"2022-11-11T01:06:19.474Z","run_id":2,"script_node":"read_order_items","batch_idx":0,"batches_total":1,"status":2,"first_token":0,"last_token":0,"comment":""}]} -->
<Common bind:this={common} />

<table>
    <thead>
        <th>Timestamp</th>
        <th>Batch idx / Total</th>
        <th>Status</th>
        <th>First token</th>
        <th>Last token</th>
        <th>Comment</th>
    </thead>
	<tbody>
        {#each webapiData.batch_history as e}
        <tr>
            <td>{dayjs(e.ts).format("MMM D, YYYY HH:mm:ss.SSS Z")}</td>
            <td>{e.batch_idx} / {e.batches_total}</td>
            <td><img src={common.nodeStatusToIcon(e.status)} title={common.nodeStatusToText(e.status)} alt=""/></td>
            <td>{e.first_token}</td>
            <td>{e.last_token}</td>
            <td>{e.comment}</td>
        </tr>
        {/each}
    </tbody>
</table>

