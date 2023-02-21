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
	let responseError = "";

    function setWebapiData(dataFromJson, errorFromJson) {
		webapiData = ( !!dataFromJson ? dataFromJson : {run_lifespans:[], nodes:[]});
		if (!!errorFromJson) {
			responseError = errorFromJson.msg
		} else {
			responseError = "";
		}

		for (var lsIdx=0; lsIdx < webapiData.run_lifespans.length; lsIdx++){
			const tsStart = dayjs(webapiData.run_lifespans[lsIdx].start_ts).valueOf();
			const tsCompleted = dayjs(webapiData.run_lifespans[lsIdx].completed_ts).valueOf();
			const tsStopped = dayjs(webapiData.run_lifespans[lsIdx].stopped_ts).valueOf();
			webapiData.run_lifespans[lsIdx].elapsed = Math.round((tsCompleted > 0 ? tsCompleted - tsStart : ( tsStopped > 0 ? tsStopped - tsStart : Date.now() - tsStart)) / 1000).toString();
		}
	}

	var timer;
	let isDestroyed = false;
	function fetchData() {
		let url = webapiUrl() + "/ks/" + params.ks_name;
		let method = "GET";
		fetch(new Request(url, {method: method}))
      		.then(response => response.json())
      		.then(responseJson => {
				handleResponse(responseJson, setWebapiData);
				if (!isDestroyed)
					timer = setTimeout(fetchData, 500);
			})
      		.catch(error => {
				responseError = method + " " + url + ":" + error;
				console.log(error);
				if (!isDestroyed)
					timer = setTimeout(fetchData, 3000);
			});
	}

	onMount(async () => {
		breadcrumbsPathElements = [{ title:"Keyspaces", link:util.rootLink() },{ title:params.ks_name }  ];
    	fetchData();
    });
	onDestroy(async () => {
		isDestroyed = true;
    	if (timer) clearTimeout(timer);
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
	thead th:not(:first-child) {text-align: center;font-size:large;}
</style>

<Util bind:this={util} />
<Breadcrumbs bind:pathElements={breadcrumbsPathElements}/>
<p style="color:red;">{responseError}</p>
<table>
	<thead>
		<th>Nodes ({webapiData.nodes.length}) \ Runs ({webapiData.run_lifespans.length})</th>
		{#each webapiData.run_lifespans as ls}
		  <th>
			<a href={util.ksRunNodeHistoryLink(params.ks_name, ls.run_id)} title="Run {ls.run_id}">
				{ls.run_id}<img src={util.runStatusToIconLink(ls.final_status)} title={util.runStatusToText(ls.final_status)} alt="" style="margin-left:3px;"/>
			</a> 
	  </th>
		{/each}
		<th><button title="Opens a popup to specify parameters (keyspace, script URI etc) for a new run" on:click={onNew}>New</button></th>
	</thead>
	<tbody>
		<tr>
			<td></td>
			{#each webapiData.run_lifespans as ls}
				<td style="font-size:small;">{ls.elapsed}s</td>
			{/each}
			<td>&nbsp;</td>
		</tr>
		<tr>
			<td></td>
			{#each webapiData.run_lifespans as ls}
				<td>{#if ls.final_status != 3}<button on:click={onStop(ls.run_id)} title={(ls.final_status === 1 ? "Stop run":"Invalidate the results of a complete run so they cannot be used in depending runs")}>{#if ls.final_status === 1}Stop{:else}Invalidate{/if}</button>{:else}&nbsp;{/if}</td>
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
