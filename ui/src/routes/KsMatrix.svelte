<script>
    import { onDestroy, onMount } from "svelte";
	import { openModal } from "svelte-modals";
	import dayjs from "dayjs";
	import Breadcrumbs from "../panels/Breadcrumbs.svelte";
    import ModalStopRun from "../modals/ModalStopRun.svelte";
	import ModalStartRun from "../modals/ModalStartRun.svelte";
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
		breadcrumbsPathElements = [{ title:"Keyspaces", link:util.rootLink() },{ title:params.ks_name }  ];
		fetchData();
		timer = setInterval(fetchData, 500);
    });
	onDestroy(async () => {
    	clearInterval(timer);
    });

    function onStop(runId) {
        openModal(ModalStopRun, { keyspace: params.ks_name, run_id: runId });
    }

    function onNew() {
        openModal(ModalStartRun, { keyspace: params.ks_name});
    }

</script>

<style>
	img { width: 20px; vertical-align: text-bottom;	}
	tr td:not(:first-child) {text-align: center;}
	thead th:not(:first-child) {text-align: center;}
</style>

<Util bind:this={util} />
<Breadcrumbs bind:pathElements={breadcrumbsPathElements}/>

<table>
	<thead>
		<th>Nodes ({webapiData.nodes.length}) \ Runs ({webapiData.run_lifespans.length})</th>
		{#each webapiData.run_lifespans as ls}
		  <th>
			<a href={util.ksRunNodeHistoryLink(params.ks_name, ls.run_id)}>
				{ls.run_id}<img src={util.runStatusToIconLink(ls.final_status)} title={util.runStatusToText(ls.final_status)} alt="" style="margin-left:3px;"/>
			</a>
		  </th>
		{/each}
		<th><button on:click={onNew}>New</button></th>
	</thead>
	<tbody>
		<tr>
			<td></td>
			{#each webapiData.run_lifespans as ls}
				<td>{#if ls.final_status != 3}<button on:click={onStop(ls.run_id)} title={(ls.final_status === 1 ? "Stop run":"Invalidate results of a complete run")}>{#if ls.final_status === 1}Stop{:else}Invalidate{/if}</button>{:else}&nbsp;{/if}</td>
			{/each}
			<td>&nbsp;</td>
		</tr>
		{#each webapiData.nodes as node}
		<tr>
			<td>{node.node_name}</td>
			{#each node.node_statuses as ns}
			<td>
				{#if ns.status > 0}
					<a href={util.ksRunNodeBatchHistoryLink(params.ks_name, ns.run_id, node.node_name)}>
						<img src={util.nodeStatusToIconLink(ns.status)} title={util.nodeStatusToText(ns.status) + " - " + dayjs(ns.ts).format()} alt=""/>
					</a>
				{/if}
			</td>
			{/each}
			<td>&nbsp;</td>
		</tr>
	  {/each}
	</tbody>
</table>
