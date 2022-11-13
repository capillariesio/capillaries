<script>
    import {afterUpdate } from 'svelte';
    import dayjs from "dayjs";
	import Util from '../Util.svelte';
	let util;

    // Component parameters
    export let node_history = [];
    export let ks_name = "";

    afterUpdate(() => {
        // Calculate elapsed times for each batch
        let nodeStartMap = {};
        for (let i=0; i < node_history.length; i++) {
            let e = node_history[i];
            if (e.status === 1) {
                nodeStartMap[e.script_node] = dayjs(e.ts).valueOf();
            }
        }

        let nodeEndMap = {};
        for (let i=0; i < node_history.length; i++) {
            let e = node_history[i];
            if (e.status > 1 && !(e.script_node in nodeEndMap)) {
                nodeEndMap[e.script_node] = dayjs(e.ts).valueOf();
            }
        }

        for (let i=0; i < node_history.length; i++) {
            let e = node_history[i];
            if (e.script_node in nodeStartMap) {
                if (e.script_node in nodeEndMap) {
                    node_history[i].elapsed = (nodeEndMap[e.script_node] - nodeStartMap[e.script_node])/1000;
                } else {
                    node_history[i].elapsed = (Date.now() - nodeStartMap[e.script_node])/1000;
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
        <th>Timestamp</th>
        <th>Node</th>
        <th>Status</th>
        <th>Elapsed</th>
        <th>Comment</th>
    </thead>
	<tbody>
        {#each node_history as e}
        <tr>
            <td style="white-space: nowrap;">{dayjs(e.ts).format("MMM D, YYYY HH:mm:ss.SSS Z")}</td>
            <td><a href={util.ksRunNodeBatchHistoryLink(ks_name, e.run_id, e.script_node)}>{e.script_node}</a></td>
            <td><img src={util.nodeStatusToIcon(e.status)} title={util.nodeStatusToText(e.status)} alt=""/></td>
            <td>{#if e.elapsed > 0} {e.elapsed} {/if}</td>
            <td>{e.comment}</td>
        </tr>
        {/each}
    </tbody>
</table>

