<script>
    import {afterUpdate } from 'svelte';
    import dayjs from "dayjs";
	import Util, {nodeStatusToColor} from '../Util.svelte';
	let util;

    // Component parameters
    export let node_history = [];
    export let ks_name = "";

    let svgSummary = "";

    afterUpdate(() => {
        let earliestTs = null;
        let latestTs = null;

        // Calculate elapsed times for each batch
        let nodeStartMap = {};
        for (let i=0; i < node_history.length; i++) {
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
        for (let i=0; i < node_history.length; i++) {
            let e = node_history[i];
            if (e.status > 1 && !(e.script_node in nodeEndMap)) {
                nodeEndMap[e.script_node] = dayjs(e.ts).valueOf();
                if (latestTs == null || nodeEndMap[e.script_node] > latestTs) {
                    latestTs = nodeEndMap[e.script_node];
                }
                nodeStatusMap[e.script_node] = e.status;
            }
        }

        let nodesTotal = Object.keys(nodeStartMap).length;
        let svgWidth = 800;
        let svgHeight = 600;
        if (earliestTs != null && latestTs != null && nodesTotal > 1) {
            svgSummary = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 ${svgWidth} ${svgHeight}" width="${svgWidth}px" height="${svgHeight}px">\n`;
            svgSummary += `<rect width="${svgWidth}" height="${svgHeight}" fill="lightgray" />`;
            let nodeElapsed = Math.round((latestTs - earliestTs) / 1000);
            let lineWidth = svgHeight / nodesTotal;
            let nodeIdx = 0;
            for (var node in nodeStartMap) {
                if (node in nodeEndMap) {
                    let startX = (nodeStartMap[node] - earliestTs) / (latestTs - earliestTs) * svgWidth;
                    let topY = nodeIdx * lineWidth;
                    let endX = (nodeEndMap[node] - earliestTs) / (latestTs - earliestTs) * svgWidth;
                    let bottomY = (nodeIdx + 1) * lineWidth;
                    svgSummary += `<path d="M${startX},${topY} L${endX},${topY} L${endX},${bottomY} L${startX},${bottomY} Z" fill="${nodeStatusToColor(nodeStatusMap[node])}" ><title>${node}</title></path>`;
                    nodeIdx++;
                }
            }
            svgSummary += '</svg>';
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
{@html svgSummary}
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
            <td><img src={util.nodeStatusToIconStatic(e.status)} title={util.nodeStatusToText(e.status)} alt=""/></td>
            <td>{#if e.elapsed > 0} {e.elapsed} {/if}</td>
            <td>{e.comment}</td>
        </tr>
        {/each}
    </tbody>
</table>

