<script>
	import Common from '../Common.svelte';
	let common;
    import {afterUpdate } from 'svelte';
    import dayjs from "dayjs";

    export let batch_history = {};

    afterUpdate(() => {
        // Calculate elapsed times for each batch
        let batchStartMap = {};
        for (let i=0; i < batch_history.length; i++) {
            let e = batch_history[i];
            if (e.status === 1) {
                batchStartMap[e.batch_idx] = dayjs(e.ts).valueOf();
            }
        }

        let batchEndMap = {};
        for (let i=0; i < batch_history.length; i++) {
            let e = batch_history[i];
            if (e.status > 1 && !(e.batch_idx in batchEndMap)) {
                batchEndMap[e.batch_idx] = dayjs(e.ts).valueOf();
            }
        }

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

<Common bind:this={common} />

<style>
    th {white-space: nowrap;}
    img {width: 20px;}
</style>
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
            <td><img src={common.nodeStatusToIcon(e.status)} title={common.nodeStatusToText(e.status)} alt=""/></td>
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

