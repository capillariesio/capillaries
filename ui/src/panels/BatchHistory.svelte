<script>
	import { afterUpdate } from 'svelte';
	import dayjs from 'dayjs';
	import Util, { nodeStatusToColor } from '../Util.svelte';
	let util;

	// Component parameters
	export let batch_history = [];

	let nodeElapsed = 0;
	let elapsedAverage = 0;
	let elapsedMedian = 0;

	let nonStartedBatchesPercent = 100;
	let nonStartedBatchesRatio = '';
	let nonStartedBatches = '';
	let startedBatchesPercent = 0;
	let startedBatchesRatio = '';
	let startedBatches = '';
	let runningBatchesPercent = 0;
	let runningBatchesRatio = '';
	let runningBatches = '';
	let finishedBatchesPercent = 0;
	let finishedBatchesRatio = '';
	let finishedBatches = '';

	let svgSummary = '';
	let executionDetailsVisible = false;

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
	afterUpdate(() => {
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
		nonStartedBatchesPercent =
			(nonStartedBatchesArray.length.toString() / batch_history[0].batches_total.toString()) * 100;
		nonStartedBatchesRatio =
			nonStartedBatchesArray.length.toString() + '/' + batch_history[0].batches_total.toString();
		nonStartedBatches = arrayToReadable(nonStartedBatchesArray);

		let startedBatchesArray = Array.from(Object.keys(batchStartMap));
		startedBatchesPercent =
			(startedBatchesArray.length.toString() / batch_history[0].batches_total.toString()) * 100;
		startedBatchesRatio =
			startedBatchesArray.length.toString() + '/' + batch_history[0].batches_total.toString();
		startedBatches = arrayToReadable(startedBatchesArray);

		let runningBatchesArray = Array.from(runningBatchSet).sort(function (a, b) {
			return a - b;
		});
		runningBatchesPercent =
			(runningBatchesArray.length.toString() / batch_history[0].batches_total.toString()) * 100;
		runningBatchesRatio =
			runningBatchesArray.length.toString() + '/' + batch_history[0].batches_total.toString();
		runningBatches = arrayToReadable(runningBatchesArray);

		let finishedBatchesArray = Array.from(Object.keys(batchEndMap));
		finishedBatchesPercent =
			(finishedBatchesArray.length.toString() / batch_history[0].batches_total.toString()) * 100;
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
			svgSummary = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 ${svgWidth} ${svgHeight}" width="${svgWidth}px" height="${svgHeight}px">\n`;
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
			svgSummary += `<text x="50" y="15" font-size="15">No timeline chart: no data yet</text>`;
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
			elapsedAverage = (elapsedSum / elapsed.length).toFixed(1);
			elapsedMedian = findMedian(elapsed).toFixed(1);
		}
	});
</script>

<Util bind:this={util} />
<table>
	<thead>
		<th colspan="2">Node batch execution summary</th>
	</thead>
	<tbody>
		<tr>
			<td>
				<!-- eslint-disable-next-line svelte/no-at-html-tags -->
				{@html svgSummary}
			</td>
			<td>
				{#if executionDetailsVisible}
					<table>
						<tbody>
							<tr>
								<td>Elapsed:</td>
								<td>{nodeElapsed}s</td>
								<td colspan="2">(not very useful, skewed by other nodes)</td>
							</tr>
							<tr>
								<td>Elapsed average:</td>
								<td colspan="3">{elapsedAverage}s</td>
							</tr>
							<tr>
								<td>Elapsed median:</td>
								<td colspan="3">{elapsedMedian}s</td>
							</tr>
							<tr>
								<td>Batches not started:</td>
								<td style="text-align: right; width:1%;">{nonStartedBatchesPercent.toFixed(0)}%</td>
								<td style="text-align: right; width:1%;">{nonStartedBatchesRatio}</td>
								<td>{nonStartedBatches}</td>
							</tr>
							<tr>
								<td>Batches started:</td>
								<td style="text-align: right; width:1%;">{startedBatchesPercent.toFixed(0)}%</td>
								<td style="text-align: right; width:1%;">{startedBatchesRatio}</td>
								<td>{startedBatches}</td>
							</tr>
							<tr>
								<td>Batches running:</td>
								<td style="text-align: right; width:1%;">{runningBatchesPercent.toFixed(0)}%</td>
								<td style="text-align: right; width:1%;">{runningBatchesRatio}</td>
								<td>{runningBatches}</td>
							</tr>
							<tr>
								<td>Batches finished:</td>
								<td style="text-align: right; width:1%;">{finishedBatchesPercent.toFixed(0)}%</td>
								<td style="text-align: right; width:1%;">{finishedBatchesRatio}</td>
								<td>{finishedBatches}</td>
							</tr>
						</tbody>
					</table>
				{/if}
			</td>
		</tr>
	</tbody>
</table>
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
		{#each batch_history as e}
			<tr>
				<td style="white-space: nowrap;">{dayjs(e.ts).format('MMM D, YYYY HH:mm:ss.SSS Z')}</td>
				<td>{e.batch_idx} / {e.batches_total}</td>
				<td
					><img
						src={util.nodeStatusToIconStatic(e.status)}
						title={util.nodeStatusToText(e.status)}
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

<style>
	th {
		white-space: nowrap;
	}
	img {
		width: 20px;
	}
</style>
