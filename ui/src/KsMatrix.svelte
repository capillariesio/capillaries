<script>
    import { onDestroy, onMount } from "svelte";
	import dayjs from "dayjs";

	import Common, { handleResponse } from './Common.svelte';
	let common;

    export let params; 

	let webapiData = {run_statuses:[], nodes:[]};
	function setWebapiData(dataFromJson) {
		webapiData = ( !!dataFromJson ? dataFromJson : {run_statuses:[], nodes:[]});
	}

	var timer;

	function fetchData() {
		fetch("http://localhost:6543/ks/"+params.ks_name)
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

	function nodeStatusToLink(node_name, ns) {
		return "/#/ks/" + params.ks_name + "/run/" + ns.run_id + "/node/" + node_name + "/batch_history";
	}

</script>

<style>
	img { width: 20px;	}
</style>

<Common bind:this={common} />

<h1>Keyspace matrix: {params.ks_name}</h1>

<table>
	<thead>
		<th>Nodes ({webapiData.nodes.length}) \ Runs ({webapiData.run_statuses.length})</th>
		{#each webapiData.run_statuses as r}
		  <th>{r.run_id} <img src={common.runStatusToIcon(r.status)} title={common.runStatusToText(r.status) + " - " + dayjs(r.ts).format()} alt=""/></th>
		{/each}
	</thead>
	<tbody>
		{#each webapiData.nodes as node}
		<tr>
			<td>{node.node_name}</td>
			{#each node.node_statuses as ns}
			<td>
				{#if ns.status > 0}
					<a href={nodeStatusToLink(node.node_name, ns)}>
						<img src={common.nodeStatusToIcon(ns.status)} title={common.nodeStatusToText(ns.status) + " - " + dayjs(ns.ts).format()} alt=""/>
					</a>
				{/if}
			</td>
			{/each}
		</tr>
	  {/each}
	</tbody>
</table>
