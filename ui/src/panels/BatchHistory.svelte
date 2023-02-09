<script>
    import {afterUpdate } from 'svelte';
    import dayjs from "dayjs";
	import Util from '../Util.svelte';
	let util;

    // Component parameters
    export let batch_history = [];

    let nonStartedBatches = "";
    let startedBatches = "";
    let runningBatches = "";
    let finishedBatches = "";
    function arrayToReadable(arr){
        var result = "";
        var prev = -1;
        for (var i=0; i < arr.length; i++) {
            if (prev == -1){
            result += arr[i].toString();
            prev = arr[i];
            continue;
            }
            
            if (prev != arr[i]-1) {
            // End of strike
            if (!result.endsWith(prev.toString())) {
                    result += prev.toString();
            }
            if (result.length > 0) {
                result += ",";
            }
            result += arr[i].toString();
            } else {
            if (!result.endsWith("-")){
                result += "-";
            }
            }
            prev = arr[i];
        }
        if (result.endsWith("-")) {
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
        for (let i=0; i < batch_history.length; i++) {
            let e = batch_history[i];
            if (e.status === 1) {
                batchStartMap[e.batch_idx] = dayjs(e.ts).valueOf();
                runningBatchSet.add(e.batch_idx)
            }
        }

        let batchEndMap = {};
        for (let i=0; i < batch_history.length; i++) {
            let e = batch_history[i];
            if (e.status > 1 && !(e.batch_idx in batchEndMap)) {
                batchEndMap[e.batch_idx] = dayjs(e.ts).valueOf();
                runningBatchSet.delete(e.batch_idx)
            }
        }

        nonStartedBatches = arrayToReadable(([...Array(batch_history[0].batches_total).keys()].filter((i) => !(i in batchStartMap))));
        startedBatches = arrayToReadable(Array.from(Object.keys(batchStartMap)));
        runningBatches = arrayToReadable(Array.from(runningBatchSet).sort(function(a, b) { return a - b;}));
        finishedBatches = arrayToReadable(Array.from(Object.keys(batchEndMap)));

        for (let i=0; i < batch_history.length; i++) {
            let e = batch_history[i];
            if (e.batch_idx in batchStartMap) {
                if (e.batch_idx in batchEndMap) {
                    batch_history[i].elapsed = (batchEndMap[e.batch_idx] - batchStartMap[e.batch_idx])/1000;
                } else {
                    batch_history[i].elapsed = (Date.now() - batchStartMap[e.batch_idx])/1000;
                }
            }
        }
	});
</script>

<Util bind:this={util} />

<style>
    th {white-space: nowrap;}
    img {width: 20px;}
</style>
<table>
    <thead>
        <th colspan="2">Batch summary</th>
    </thead>
    <tbody>
        <tr>
            <td>Not started:</td>
            <td>{nonStartedBatches}</td>
        </tr>
        <tr>
            <td>Started:</td>
            <td>{startedBatches}</td>
        </tr>
        <tr>
            <td>Running:</td>
            <td>{runningBatches}</td>
        </tr>
        <tr>
            <td>Finished:</td>
            <td>{finishedBatches}</td>
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
            <td style="white-space: nowrap;">{dayjs(e.ts).format("MMM D, YYYY HH:mm:ss.SSS Z")}</td>
            <td>{e.batch_idx} / {e.batches_total}</td>
            <td><img src={util.nodeStatusToIconStatic(e.status)} title={util.nodeStatusToText(e.status)} alt=""/></td>
            <td>{#if e.elapsed > 0} {e.elapsed} {/if}</td>
            <td>{e.first_token}</td>
            <td>{e.last_token}</td>
            <td>{e.instance}</td>
            <td>{e.thread}</td>
            <td>{e.comment}</td>
        </tr>
        {/each}
    </tbody>
</table>

