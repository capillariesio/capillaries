<script>
    import { onDestroy, onMount } from "svelte";
	import dayjs from "dayjs";
	import Breadcrumbs from "../panels/Breadcrumbs.svelte";
	import Util, { webapiUrl, handleResponse } from '../Util.svelte';
	let util;

    // Route params
    export let params; 

	// Breadcrumbs
	let breadcrumbsPathElements = [];

    // Webapi data
	let webapiData = {run_lifespans:[], nodes:[]};
	function setWebapiData(dataFromJson) {
		webapiData = ( !!dataFromJson ? dataFromJson : {run_lifespans:[], nodes:[]});
	}

	function fetchData() {
		fetch(webapiUrl() + "/ks/" + params.ks_name)
      		.then(response => response.json())
      		.then(responseJson => { handleResponse(responseJson, setWebapiData);})
      		.catch(error => {console.log(error);});
	}

	var timer;
	onMount(async () => {
		breadcrumbsPathElements = [{ title:"Keyspaces", link:util.rootLink() },{ title:params.ks_name + " matrix" }  ];
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

<Util bind:this={util} />
<Breadcrumbs bind:pathElements={breadcrumbsPathElements}/>

<table>
	<thead>
		<th>Nodes ({webapiData.nodes.length}) \ Runs ({webapiData.run_lifespans.length})</th>
		{#each webapiData.run_lifespans as ls}
		  <th>
			{ls.run_id}
			<a href={util.ksRunNodeHistoryLink(params.ks_name, ls.run_id)}>
				<img src={util.runStatusToIcon(ls.final_status)} title={util.runStatusToText(ls.final_status)} alt=""/>
			</a>
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
					<a href={util.ksRunNodeBatchHistoryLink(params.ks_name, ns.run_id, node.node_name)}>
						<img src={util.nodeStatusToIcon(ns.status)} title={util.nodeStatusToText(ns.status) + " - " + dayjs(ns.ts).format()} alt=""/>
					</a>
				{/if}
			</td>
			{/each}
		</tr>
	  {/each}
	</tbody>
</table>
