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
		nodeStatusToIconStatic,
		nodeStatusToText,
		nodeStatusToColor
	} from '../Util.svelte';

	const { ks_name, run_id, node_name } = $props();

	let breadcrumbsPathElements = $state([]);
	let webapiData = $state({ run_props: {}, run_lifespan: {}, batch_history: [] });
	let responseError = $state('');
	var timer;
	let isDestroyed = false;

	function setWebapiData(dataFromJson, errorFromJson) {
		if (dataFromJson) {
			webapiData.run_props = dataFromJson.run_props;
			webapiData.run_lifespan = dataFromJson.run_lifespan;
			webapiData.batch_history = dataFromJson.batch_history;
			calculateElapsed(webapiData.batch_history);
		}
		if (errorFromJson) {
			responseError =
				'cannot retrieve batch history, Capillaries webapi returned an error: ' + errorFromJson;
		} else {
			responseError = '';
		}
	}

	function fetchData() {
		let url =
			webapiUrl() + '/ks/' + ks_name + '/run/' + run_id + '/node/' + node_name + '/batch_history';
		let method = 'GET';
		fetch(new Request(url, { method: method }))
			.then((response) => response.json())
			.then((responseJson) => {
				handleResponse(responseJson, setWebapiData);
				if (!isDestroyed) {
					if (webapiData.run_lifespan.final_status > 1) {
						// Run complete, nothing to expect here
						timer = setTimeout(fetchData, 5000);
					} else {
						timer = setTimeout(fetchData, 500);
					}
				}
			})
			.catch((error) => {
				responseError =
					'cannot fetch batch history data from Capillaries webapi at ' +
					method +
					' ' +
					url +
					', error:' +
					error;
				console.log(error);
				if (!isDestroyed) timer = setTimeout(fetchData, 3000);
			});
	}

	onMount(() => {
		breadcrumbsPathElements = [
			{ title: 'Keyspaces', link: rootLink() },
			{ title: ks_name, link: ksMatrixLink(ks_name) },
			{ title: 'run ' + run_id},
			{ title: 'node ' + node_name },
			{ title: 'batch processing'}
		];
		fetchData();
	});
	onDestroy(() => {
		isDestroyed = true;
		if (timer) clearTimeout(timer);
	});

	let nodeElapsed = $state(0);
	let elapsedAverage = $state(0);
	let elapsedMedian = $state(0);
	let elapsedStandardDeviation = $state(0);
	let elapsedCVPercent = $state(0);

	let nonStartedBatchesPercent = $state(100);
	let nonStartedBatchesRatio = $state('');
	let nonStartedBatches = $state('');
	let startedBatchesPercent = $state(0);
	let startedBatchesRatio = $state('');
	let startedBatches = $state('');
	let runningBatchesPercent = $state(0);
	let runningBatchesRatio = $state('');
	let runningBatches = $state('');
	let finishedBatchesPercent = $state(0);
	let finishedBatchesRatio = $state('');
	let finishedBatches = $state('');

	let svgSummary = $state('');
	let executionDetailsVisible = $state(false);

	function findMedian(arr) {
		arr.sort((a, b) => a - b);
		const middleIndex = Math.floor(arr.length / 2);

		if (arr.length % 2 === 0) {
			return (arr[middleIndex - 1] + arr[middleIndex]) / 2;
		} else {
			return arr[middleIndex];
		}
	}

	function arrayToReadable(arr) {
		var result = '';
		var prev = -1;
		for (var i = 0; i < arr.length; i++) {
			if (prev == -1) {
				result += arr[i].toString();
				prev = arr[i];
				continue;
			}

			if (prev != arr[i] - 1) {
				// End of strike
				if (!result.endsWith(prev.toString())) {
					result += prev.toString();
				}
				if (result.length > 0) {
					result += ',';
				}
				result += arr[i].toString();
			} else {
				if (!result.endsWith('-')) {
					result += '-';
				}
			}
			prev = arr[i];
		}
		if (result.endsWith('-')) {
			result += prev.toString();
		}
		return result;
	}

	function trimZeroAndHundred(val){
		if (val > 0 && val < 1.0){
			return 1.0;
		} else if (val > 99 && val < 100){
			return 99.0;
		}
		else return val;
	}

	function calculateElapsed(batch_history) {
		if (batch_history.length == 0) {
			return;
		}
		// Calculate elapsed times for each batch
		let runningBatchSet = new Set();
		let batchStartMap = {};
		let earliestTs = null;
		let latestTs = null;
		for (let i = 0; i < batch_history.length; i++) {
			let e = batch_history[i];
			if (e.status === 1) {
				batchStartMap[e.batch_idx] = dayjs(e.ts).valueOf();
				if (earliestTs == null || batchStartMap[e.batch_idx] < earliestTs) {
					earliestTs = batchStartMap[e.batch_idx];
				}
				runningBatchSet.add(e.batch_idx);
			}
		}

		let batchEndMap = {};
		let batchStatusMap = {};
		for (let i = 0; i < batch_history.length; i++) {
			let e = batch_history[i];
			if (e.status > 1 && !(e.batch_idx in batchEndMap)) {
				batchEndMap[e.batch_idx] = dayjs(e.ts).valueOf();
				if (latestTs == null || batchEndMap[e.batch_idx] > latestTs) {
					latestTs = batchEndMap[e.batch_idx];
				}
				batchStatusMap[e.batch_idx] = e.status;
				runningBatchSet.delete(e.batch_idx);
			}
		}

		let nonStartedBatchesArray = [...Array(batch_history[0].batches_total).keys()].filter(
			(i) => !(i in batchStartMap)
		);
		nonStartedBatchesPercent = trimZeroAndHundred((nonStartedBatchesArray.length.toString() / batch_history[0].batches_total.toString()) * 100);
		nonStartedBatchesRatio =
			nonStartedBatchesArray.length.toString() + '/' + batch_history[0].batches_total.toString();
		nonStartedBatches = arrayToReadable(nonStartedBatchesArray);

		let startedBatchesArray = Array.from(Object.keys(batchStartMap));
		startedBatchesPercent = trimZeroAndHundred(
			(startedBatchesArray.length.toString() / batch_history[0].batches_total.toString()) * 100);
		startedBatchesRatio =
			startedBatchesArray.length.toString() + '/' + batch_history[0].batches_total.toString();
		startedBatches = arrayToReadable(startedBatchesArray);

		let runningBatchesArray = Array.from(runningBatchSet).sort(function (a, b) {
			return a - b;
		});
		runningBatchesPercent = trimZeroAndHundred(
			(runningBatchesArray.length.toString() / batch_history[0].batches_total.toString()) * 100);
		runningBatchesRatio =
			runningBatchesArray.length.toString() + '/' + batch_history[0].batches_total.toString();
		runningBatches = arrayToReadable(runningBatchesArray);

		let finishedBatchesArray = Array.from(Object.keys(batchEndMap));
		finishedBatchesPercent = trimZeroAndHundred(
			(finishedBatchesArray.length.toString() / batch_history[0].batches_total.toString()) * 100);
		finishedBatchesRatio =
			finishedBatchesArray.length.toString() + '/' + batch_history[0].batches_total.toString();
		finishedBatches = arrayToReadable(finishedBatchesArray);

		if (earliestTs != null && latestTs != null && batch_history[0].batches_total > 1) {
			executionDetailsVisible = true;
			let svgWidth = 800;
			let svgHeight = 1200; // Max height
			let lineWidth = 10;
			if (lineWidth * batch_history[0].batches_total < svgHeight) {
				svgHeight = lineWidth * batch_history[0].batches_total;
			} else {
				lineWidth = svgHeight / batch_history[0].batches_total;
			}
			//svgSummary = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 ${svgWidth} ${svgHeight}" width="${svgWidth}px" height="${svgHeight}px">\n`;
			svgSummary = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 ${svgWidth} ${svgHeight}">\n`;
			svgSummary += `<rect width="${svgWidth}" height="${svgHeight}" fill="lightgray" />`;
			nodeElapsed = Math.round((latestTs - earliestTs) / 1000);
			for (var batchIdx = 0; batchIdx < batch_history[0].batches_total; batchIdx++) {
				if (batchIdx in batchStartMap && batchIdx in batchEndMap) {
					let startX =
						((batchStartMap[batchIdx] - earliestTs) / (latestTs - earliestTs)) * svgWidth;
					let topY = batchIdx * lineWidth;
					let endX = ((batchEndMap[batchIdx] - earliestTs) / (latestTs - earliestTs)) * svgWidth;
					let bottomY = (batchIdx + 1) * lineWidth;
					svgSummary += `<path d="M${startX},${topY} L${endX},${topY} L${endX},${bottomY} L${startX},${bottomY} Z" fill="${nodeStatusToColor(
						batchStatusMap[batchIdx]
					)}" ><title>Batch ${batchIdx} ${(
						(batchEndMap[batchIdx] - batchStartMap[batchIdx]) /
						1000
					).toFixed(1)}s</title></path>`;
				}
			}
			svgSummary += '</svg>';
		} else {
			svgSummary = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 400 50" width="400px" height="50px">\n`;
			svgSummary += `<text x="50" y="15" font-size="15">No timeline chart: no completed batches yet</text>`;
			svgSummary += `<text x="10" y="35" font-size="15">or the number of batches does not exceed 1</text>`;
			svgSummary += '</svg>';
		}

		var elapsed = [];
		var elapsedSum = 0;
		for (let i = 0; i < batch_history.length; i++) {
			let e = batch_history[i];
			if (e.batch_idx in batchStartMap) {
				if (e.batch_idx in batchEndMap) {
					batch_history[i].elapsed = (batchEndMap[e.batch_idx] - batchStartMap[e.batch_idx]) / 1000;
					elapsed.push(batch_history[i].elapsed);
					elapsedSum += batch_history[i].elapsed;
				} else {
					batch_history[i].elapsed = (Date.now() - batchStartMap[e.batch_idx]) / 1000;
				}
			}
		}

		if (elapsed.length > 0) {
			elapsedAverage = (elapsedSum / elapsed.length);
			elapsedMedian = findMedian(elapsed);

			let deviationSum = 0;
			for (let i = 0; i < batch_history.length; i++) {
				let e = batch_history[i].elapsed;
				deviationSum += (e-elapsedAverage)*(e-elapsedAverage)
			}
			let variance = deviationSum/elapsed.length;
			elapsedStandardDeviation = Math.sqrt(variance);
			elapsedCVPercent = (elapsedStandardDeviation/elapsedAverage*100.0);
		}
	}
	let tabs = [
		{ label: "Batch processing summary",
		 value: 1,
		 component: tabBatchProcessingSummary
		},
		{ label: "Batch processing history",
		 value: 2,
		 component: tabBatchProcessingHistory
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

{#snippet tabBatchProcessingSummary()}
{#if executionDetailsVisible}
<table>
	<tbody>
		<tr>
			<td style="width:250px">Elapsed total:</td>
			<td style="width:150px; text-align: right;">{nodeElapsed}s</td>
			<td colspan="2">(not very useful, usually skewed by other nodes/runs/scripts)</td>
		</tr>
		<tr>
			<td>Elapsed average:</td>
			<td style="text-align: right;">{elapsedAverage.toFixed(1)}s</td>
			<td colspan="2"></td>
		</tr>
		<tr>
			<td>Elapsed standard deviation and CV:</td>
			<td style="text-wrap-mode: nowrap;text-align: right;">{elapsedStandardDeviation.toFixed(2)}s ({elapsedCVPercent.toFixed(1)}%)</td>
			<td colspan="2">(small SD means no data skew and even distribution of the CPU load)</td>
		</tr>
		<tr>
			<td>Elapsed median:</td>
			<td style="text-align: right;">{elapsedMedian.toFixed(1)}s</td>
			<td colspan="2"></td>
		</tr>
		<tr>
			<td>Batches not started:</td>
			<td style="text-align: right;">{nonStartedBatchesPercent.toFixed(0)}%</td>
			<td style="text-align: right; width:100px;">{nonStartedBatchesRatio}</td>
			<td style="text-align: right; ">{nonStartedBatches}</td>
		</tr>
		<tr>
			<td>Batches started:</td>
			<td style="text-align: right;">{startedBatchesPercent.toFixed(0)}%</td>
			<td style="text-align: right;">{startedBatchesRatio}</td>
			<td style="text-align: right;">{startedBatches}</td>
		</tr>
		<tr>
			<td>Batches running:</td>
			<td style="text-align: right;">{runningBatchesPercent.toFixed(0)}%</td>
			<td style="text-align: right;">{runningBatchesRatio}</td>
			<td style="text-align: right;">{runningBatches}</td>
		</tr>
		<tr>
			<td>Batches finished:</td>
			<td style="text-align: right;">{finishedBatchesPercent.toFixed(0)}%</td>
			<td style="text-align: right;">{finishedBatchesRatio}</td>
			<td style="text-align: right;">{finishedBatches}</td>
		</tr>
	</tbody>
</table>
{/if}
<div style="width:100%">
	<!-- eslint-disable-next-line svelte/no-at-html-tags -->
	{@html svgSummary}
</div>
{/snippet}

{#snippet tabBatchProcessingHistory()}
<table>
	<thead>
		<tr>
			<th>Timestamp</th>
			<th>Batch</th>
			<th>Status</th>
			<th>Elapsed</th>
			<th>First token</th>
			<th>Last token</th>
			<th>Host/Instance</th>
			<th>Thread</th>
			<th>Comment</th>
		</tr>
	</thead>
	<tbody>
		{#each webapiData.batch_history as e}
			<tr>
				<td style="white-space: nowrap;">{dayjs(e.ts).format('MMM D, YYYY HH:mm:ss.SSS Z')}</td>
				<td>{e.batch_idx} / {e.batches_total}</td>
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
				<td>{e.first_token}</td>
				<td>{e.last_token}</td>
				<td>{e.instance}</td>
				<td>{e.thread}</td>
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
	td {
		padding: 2px;
	}
	img {
		width: 20px;
	}
</style>
