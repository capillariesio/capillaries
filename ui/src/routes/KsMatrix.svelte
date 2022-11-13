<script>
    import { onDestroy, onMount } from "svelte";
	import dayjs from "dayjs";

	import Breadcrumbs from "../panels/Breadcrumbs.svelte";
	let breadcrumbsPathElements = [];
	
	import Common, { handleResponse } from '../Common.svelte';
	let common;

    export let params; 

	let webapiData = {run_lifespans:[], nodes:[]};
	function setWebapiData(dataFromJson) {
		webapiData = ( !!dataFromJson ? dataFromJson : {run_lifespans:[], nodes:[]});
	}

	var timer;

	function fetchData() {
		fetch("http://localhost:6543/ks/"+params.ks_name)
      		.then(response => response.json())
      		.then(responseJson => { handleResponse(responseJson, setWebapiData);})
      		.catch(error => {console.log(error);});
	}

	onMount(async () => {
		breadcrumbsPathElements = [{ title:"Keyspaces", link:common.rootLink() },{ title:params.ks_name + " matrix" }  ];
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

<Common bind:this={common} />
<Breadcrumbs bind:pathElements={breadcrumbsPathElements}/>

<table>
	<thead>
		<th>Nodes ({webapiData.nodes.length}) \ Runs ({webapiData.run_lifespans.length})</th>
		{#each webapiData.run_lifespans as ls}
		  <th>
			{ls.run_id} <img src={common.runStatusToIcon(ls.final_status)} title={common.runStatusToText(ls.final_status)} alt=""/>
		  </th>
		{/each}
	</thead>
	<tbody>
		{#each webapiData.nodes as node}
		<tr>
			<td>{node.node_name}</td>
			{#each node.node_statuses as ns}
			<td>
				{#if ns.status > 0}
					<a href={common.ksRunNodeBatchHistoryLink(params.ks_name, ns.run_id, node.node_name)}>
						<img src={common.nodeStatusToIcon(ns.status)} title={common.nodeStatusToText(ns.status) + " - " + dayjs(ns.ts).format()} alt=""/>
					</a>
				{/if}
			</td>
			{/each}
		</tr>
	  {/each}
	</tbody>
</table>
