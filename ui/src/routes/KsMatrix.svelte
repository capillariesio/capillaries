<script>
	import { onDestroy, onMount } from 'svelte';
	import { modals } from 'svelte-modals';
	import dayjs from 'dayjs';
	import Breadcrumbs from '../panels/Breadcrumbs.svelte';
	import ModalStopRun from '../modals/ModalStopRun.svelte';
	import ModalStartRun from '../modals/ModalStartRun.svelte';
	import {
		webapiUrl,
		handleResponse,
		ksRunNodeHistoryLink,
		runStatusToIconLink,
		runStatusToText,
		ksRunNodeBatchHistoryLink,
		nodeStatusToIconLink,
		nodeStatusToText,
		rootLink,
		scriptVizUrl
	} from '../Util.svelte';

	const { ks_name } = $props();

	let breadcrumbsPathElements = $state([]);

	let svgScriptViz = $state('');
	let webapiData = $state({ run_lifespans: [], nodes: [] });
	let responseError = $state('');
	var timer;
	let isDestroyed = false;

	function setWebapiData(dataFromJson, errorFromJson) {
		webapiData = dataFromJson ? dataFromJson : { run_lifespans: [], nodes: [] };
		if (errorFromJson) {
			responseError =
				'cannot retrieve keyspace matrix, Capillaries webapi returned an error: ' + errorFromJson;
		} else {
			responseError = '';
		}

		for (var lsIdx = 0; lsIdx < webapiData.run_lifespans.length; lsIdx++) {
			const tsStart = dayjs(webapiData.run_lifespans[lsIdx].start_ts).valueOf();
			const tsCompleted = dayjs(webapiData.run_lifespans[lsIdx].completed_ts).valueOf();
			const tsStopped = dayjs(webapiData.run_lifespans[lsIdx].stopped_ts).valueOf();
			webapiData.run_lifespans[lsIdx].elapsed = Math.round(
				(tsCompleted > 0
					? tsCompleted - tsStart
					: tsStopped > 0
						? tsStopped - tsStart
						: Date.now() - tsStart) / 1000
			).toString();
		}
	}

	function fetchData() {
		let url = webapiUrl() + '/ks/' + ks_name;
		let method = 'GET';
		fetch(new Request(url, { method: method }))
			.then((response) => response.json())
			.then((responseJson) => {
				handleResponse(responseJson, setWebapiData);
				if (svgScriptViz == '' && webapiData.run_lifespans.length > 0) {
					fetchScriptViz();
				}
				if (!isDestroyed) timer = setTimeout(fetchData, 500);
			})
			.catch((error) => {
				responseError =
					'cannot fetch keyspace matrix data from Capillaries webapi at ' +
					method +
					' ' +
					url +
					', error:' +
					error;
				console.log(error);
				if (!isDestroyed) timer = setTimeout(fetchData, 3000);
			});
	}

	function fetchScriptViz() {
		let url = scriptVizUrl(ks_name, webapiData.run_lifespans[0].run_id, false); // Get B&W viz
		let method = 'GET';
		fetch(new Request(url, { method: method }))
			.then((response) => response.text())
			.then((responseText) => {
				svgScriptViz = responseText;
			})
			.catch((error) => {
				responseError =
					'cannot fetch SVG viz from Capillaries webapi at ' +
					method +
					' ' +
					url +
					', error:' +
					error;
				console.log(error);
			});
	}

	onMount(() => {
		breadcrumbsPathElements = [{ title: 'Keyspaces', link: rootLink() }, { title: ks_name }];
		fetchData();
	});
	onDestroy(() => {
		isDestroyed = true;
		if (timer) clearTimeout(timer);
	});

	function onStop(runId) {
		modals.open(ModalStopRun, { ks_name: ks_name, run_id: runId });
	}

	function onNew() {
		modals.open(ModalStartRun, { ks_name: ks_name });
	}
</script>

<Breadcrumbs path_elements={breadcrumbsPathElements} />
<p style="color:red;">{responseError}</p>
<table>
	<thead>
		<tr>
			<th>Nodes ({webapiData.nodes.length}) \ Runs ({webapiData.run_lifespans.length})</th>
			{#each webapiData.run_lifespans as ls}
				<th>
					<a href={ksRunNodeHistoryLink(ks_name, ls.run_id)} title="Run {ls.run_id}">
						{ls.run_id}
						<br />
						<img
							src={runStatusToIconLink(ls.final_status)}
							title={runStatusToText(ls.final_status)}
							alt=""
							style="margin-left:0px;"
						/>
					</a>
				</th>
			{/each}
			<th
				><button
					title="Opens a popup to specify parameters (keyspace, script URL etc) for a new run"
					onclick={onNew}>New</button
				>
			</th>
		</tr>
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
				<td
					>{#if ls.final_status != 3}<button
							onclick={() => {
								onStop(ls.run_id);
							}}
							title={ls.final_status === 1
								? 'Stop run'
								: 'Invalidate the results of a complete run so they cannot be used in depending runs'}
							>{#if ls.final_status === 1}Stop{:else}Invalidate{/if}</button
						>{:else}&nbsp;{/if}</td
				>
			{/each}
			<td>&nbsp;</td>
		</tr>
		{#each webapiData.nodes as node}
			<tr>
				<td>{node.node_name}: {node.node_desc}</td>
				{#each node.node_statuses as ns}
					<td>
						{#if ns.status > 0}
							<a href={ksRunNodeBatchHistoryLink(ks_name, ns.run_id, node.node_name)}>
								<img
									src={nodeStatusToIconLink(ns.status)}
									title={nodeStatusToText(ns.status) + ' - ' + dayjs(ns.ts).format()}
									alt=""
								/>
							</a>
						{/if}
					</td>
				{/each}
				<td>&nbsp;</td>
			</tr>
		{/each}
	</tbody>
</table>
{#if webapiData.run_lifespans.length > 0}
	<p>
		Script diagram. It's static and does not depend on the status of any run. To see it in a
		separate window, click <a
			target="_blank"
			href={scriptVizUrl(ks_name, webapiData.run_lifespans[0].run_id, false)}>here</a
		>
		for black and white, click
		<a target="_blank" href={scriptVizUrl(ks_name, webapiData.run_lifespans[0].run_id, true)}
			>here</a
		> for colored by root node.
	</p>
{/if}
<div style="width:100%">
	<!-- eslint-disable-next-line svelte/no-at-html-tags -->
	{@html svgScriptViz}
</div>

<style>
	img {
		width: 20px;
		vertical-align: text-bottom;
	}
	tr td:not(:first-child) {
		text-align: center;
	}
	thead th:not(:first-child) {
		text-align: center;
		font-size: large;
	}
</style>
