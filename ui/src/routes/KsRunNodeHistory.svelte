<script>
	import Tabs from "../Tabs.svelte";
	import dayjs from 'dayjs';
	import { onDestroy, onMount } from 'svelte';
	import RunInfo from '../panels/RunInfo.svelte';
	import Breadcrumbs from '../panels/Breadcrumbs.svelte';
	import {
		webapiUrl,
		handleResponse,
		rootLink,
		ksMatrixLink,
		statusVizUrl,
		scriptVizUrl,
		ksRunNodeBatchHistoryLink,
		nodeStatusToIconStatic,
		nodeStatusToText
	} from '../Util.svelte';

	const { ks_name, run_id } = $props();

	// Breadcrumbs
	let breadcrumbsPathElements = $state([]);

	// Webapi data
	let webapiData = $state({ run_props: {}, run_lifespan: {}, node_history: [] });
	let responseError = $state('');
	let svgStatusViz = $state('');

	function calculateElapsed(node_history) {
		let earliestTs = null;
		let latestTs = null;

		// Calculate elapsed times for each batch
		let nodeStartMap = {};
		for (let i = 0; i < node_history.length; i++) {
			let e = node_history[i];
			if (e.status === 1) {
				nodeStartMap[e.script_node] = dayjs(e.ts).valueOf();
				if (earliestTs == null || nodeStartMap[e.script_node] < earliestTs) {
					earliestTs = nodeStartMap[e.script_node];
				}
			}
		}

		let nodeEndMap = {};
		let nodeStatusMap = {};
		for (let i = 0; i < node_history.length; i++) {
			let e = node_history[i];
			if (e.status > 1 && !(e.script_node in nodeEndMap)) {
				nodeEndMap[e.script_node] = dayjs(e.ts).valueOf();
				if (latestTs == null || nodeEndMap[e.script_node] > latestTs) {
					latestTs = nodeEndMap[e.script_node];
				}
				nodeStatusMap[e.script_node] = e.status;
			}
		}

		//let nodesTotal = Object.keys(nodeStartMap).length;
		// if (earliestTs != null && latestTs != null && nodesTotal > 1) {
		// 	let svgWidth = 800;
		// 	let svgHeight = 600; // Max height
		// 	let lineWidth = 10;
		// 	if (lineWidth * nodesTotal < svgHeight) {
		// 		svgHeight = lineWidth * nodesTotal;
		// 	} else {
		// 		lineWidth = svgHeight / nodesTotal;
		// 	}
		// 	svgSummary = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 ${svgWidth} ${svgHeight}" width="${svgWidth}px" height="${svgHeight}px">\n`;
		// 	svgSummary += `<rect width="${svgWidth}" height="${svgHeight}" fill="lightgray" />`;
		// 	let nodeIdx = 0;
		// 	for (var node in nodeStartMap) {
		// 		if (node in nodeEndMap) {
		// 			let startX = ((nodeStartMap[node] - earliestTs) / (latestTs - earliestTs)) * svgWidth;
		// 			let topY = nodeIdx * lineWidth;
		// 			let endX = ((nodeEndMap[node] - earliestTs) / (latestTs - earliestTs)) * svgWidth;
		// 			let bottomY = (nodeIdx + 1) * lineWidth;
		// 			svgSummary += `<path d="M${startX},${topY} L${endX},${topY} L${endX},${bottomY} L${startX},${bottomY} Z" fill="${nodeStatusToColor(
		// 				nodeStatusMap[node]
		// 			)}" ><title>${node} ${Math.ceil(
		// 				(nodeEndMap[node] - nodeStartMap[node]) / 1000
		// 			).toString()}s</title></path>`;
		// 			nodeIdx++;
		// 		}
		// 	}
		// 	svgSummary += '</svg>';
		// }

		for (let i = 0; i < node_history.length; i++) {
			let e = node_history[i];
			if (e.script_node in nodeStartMap) {
				if (e.script_node in nodeEndMap) {
					node_history[i].elapsed =
						(nodeEndMap[e.script_node] - nodeStartMap[e.script_node]) / 1000;
				} else {
					node_history[i].elapsed = (Date.now() - nodeStartMap[e.script_node]) / 1000;
				}
			}
		}
	}

	function setWebapiData(dataFromJson, errorFromJson) {
		if (dataFromJson) {
			webapiData.run_props = dataFromJson.run_props;
			webapiData.run_lifespan = dataFromJson.run_lifespan;
			webapiData.node_history = dataFromJson.node_history;
			calculateElapsed(webapiData.node_history);
		}
		if (errorFromJson) {
			responseError =
				'cannot retrieve node history, Capillaries webapi returned an error: ' + errorFromJson;
		} else {
			responseError = '';
		}
	}

	var dataTimer;
	var svgTimer;
	let isDestroyed = false;

	function fetchData() {
		let url = webapiUrl() + '/ks/' + ks_name + '/run/' + run_id + '/node_history';
		let method = 'GET';
		fetch(new Request(url, { method: method }))
			.then((response) => response.json())
			.then((responseJson) => {
				handleResponse(responseJson, setWebapiData);
				if (!isDestroyed) {
					if (webapiData.run_lifespan.final_status > 1) {
						// Run complete, nothing to expect here
						dataTimer = setTimeout(fetchData, 10000);
					} else {
						dataTimer = setTimeout(fetchData, 1000);
					}
				}
			})
			.catch((error) => {
				responseError =
					'cannot fetch node history data from Capillaries webapi at ' +
					method +
					' ' +
					url +
					', error:' +
					error;
				console.log(error);
				dataTimer = setTimeout(fetchData, 3000);
			});
	}

	function fetchSvg() {
		let url = statusVizUrl(ks_name, run_id);
		let method = 'GET';
		fetch(new Request(url, { method: method }))
			.then((response) => response.text())
			.then((responseText) => {
				svgStatusViz = responseText;
				if (!isDestroyed) {
					if (webapiData.run_lifespan.final_status > 1) {
						// Run complete, nothing to expect here
						svgTimer = setTimeout(fetchSvg, 10000);
					} else {
						svgTimer = setTimeout(fetchSvg, 1000);
					}
				}
			})
			.catch((error) => {
				responseError =
					'cannot fetch SVG status viz from Capillaries webapi at ' +
					method +
					' ' +
					url +
					', error:' +
					error;
				console.log(error);
				svgTimer = setTimeout(fetchSvg, 3000);
			});
	}

	onMount(() => {
		breadcrumbsPathElements = [
			{ title: 'Keyspaces', link: rootLink() },
			{ title: ks_name, link: ksMatrixLink(ks_name) },
			{ title: 'run ' + run_id},
			{ title: 'node processing'}
		];
		fetchData();
		fetchSvg();
	});
	onDestroy(() => {
		isDestroyed = true;
		if (dataTimer) clearTimeout(dataTimer);
		if (svgTimer) clearTimeout(svgTimer);
	});

	let tabs = [
		{ label: "Node status diagram",
		 value: 1,
		 component: tabDiagram
		},
		{ label: "Node processing history",
		 value: 2,
		 component: tabHistory
		},
		{ label: "Run info",
		 value: 3,
		 component: tabRunInfo
		}
	];
</script>

<Breadcrumbs path_elements={breadcrumbsPathElements} />
<p style="color:red;">{responseError}</p>

{#snippet tabRunInfo()}
<RunInfo run_lifespan={webapiData.run_lifespan} run_props={webapiData.run_props} {ks_name} />
{/snippet}

{#snippet tabDiagram()}
<p>
	This diagram is dynamic and evolves as nodes are processed. Color legend:
	<span class="badge started">node started</span>,
	<span class="badge success">node completed successfully</span>,
	<span class="badge failed">node failed</span>,
	<span class="badge stopped">stop signal received</span>
	<span class="badge notstarted">node not processed as part of this run (maybe yet)</span>.
	Nodes that require manual start are marked with a thicker border.
	To see a static copy of it in a separate window,
	click <a target="_blank" href={statusVizUrl(ks_name, run_id)}>here</a>.
	To see detailed script diagram not reflecting run status,
	click <a target="_blank" href={scriptVizUrl(ks_name, run_id, false)}>here</a> for black and white,
	or <a target="_blank" href={scriptVizUrl(ks_name, run_id, true)}>here</a> for colored by root node.
</p>

<div style="width:100%">
	<!-- eslint-disable-next-line svelte/no-at-html-tags -->
	{@html svgStatusViz}
</div>
{/snippet}

{#snippet tabHistory()}
<table>
	<thead>
		<tr>
			<th>Timestamp</th>
			<th>Node</th>
			<th>Status</th>
			<th>Elapsed</th>
			<th>Comment</th>
		</tr>
	</thead>
	<tbody>
		{#each webapiData.node_history as e}
			<tr>
				<td style="white-space: nowrap;">{dayjs(e.ts).format('MMM D, YYYY HH:mm:ss.SSS Z')}</td>
				<td
					><a href={ksRunNodeBatchHistoryLink(ks_name, e.run_id, e.script_node)}>{e.script_node}</a
					></td
				>
				<td
					><img
						src={nodeStatusToIconStatic(e.status)}
						title={nodeStatusToText(e.status)}
						alt=""
					/></td
				>
				<td
					>{#if e.elapsed > 0}
						{e.elapsed}
					{/if}</td
				>
				<td>{e.comment}</td>
			</tr>
		{/each}
	</tbody>
</table>
{/snippet}

<Tabs items={tabs}/>

<style>
	th {
		white-space: nowrap;
	}
	img {
		width: 20px;
	}
	.badge {
		display: inline-block;
		padding: .25em .4em;
		font-size: 75%;
		font-weight: 400;
		line-height: 1;
		text-align: center;
		white-space: nowrap;
		white-space-collapse: collapse;
		text-wrap-mode: nowrap;
		vertical-align: baseline;
		border-radius: .25rem;
		border-width: thin;
		border-style: solid;
	}
	.success {
		border-color:#008000;
		background-color:#00800054;
	}
	.started {
		border-color:#0000FF;
		background-color:#0000FF54;
	}
	.failed {
		border-color:#FF0000;
		background-color:#FF000054;
	}
	.stopped {
		border-color:#FF8C00;
		background-color:#FF8C0054;
	}
	.notstarted {
		border-color:#000000;
		background-color:#FFFFFF54;
	}
</style>
