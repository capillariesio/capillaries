<script>
    import { openModal } from "svelte-modals";
    import dayjs from "dayjs";
    import ModalStopRun from "../modals/ModalStopRun.svelte";
    import Util from "../Util.svelte";
    let util;

    // Component parameters
    export let run_lifespan = {};
    export let run_props = {};
    export let ks_name = "";

    function onStop() {
        openModal(ModalStopRun, { keyspace: ks_name, run_id: run_lifespan.run_id });
    }

    function calculateElapsed(ls) {
        if (dayjs(ls.completed_ts).valueOf() > 0) {
            return ( (dayjs(ls.completed_ts).valueOf() - dayjs(ls.start_ts).valueOf()) / 1000);
        } else if (dayjs(ls.stopped_ts).valueOf() > 0) {
            return ( (dayjs(ls.stopped_ts).valueOf() - dayjs(ls.start_ts).valueOf()) / 1000 );
        } else {
            return (Date.now() - dayjs(ls.start_ts).valueOf()) / 1000;
        }
    }
</script>

<style>
    tr td:first-child {white-space: nowrap;}
    img {width: 20px; vertical-align: text-bottom;}
</style>

<Util bind:this={util} />

{#if Object.keys(run_lifespan).length > 0}
<table>
    <tbody>
        <tr>
            <td>Run Id:</td>
            <td>{run_lifespan.run_id}</td>
            <td></td>
        </tr>
        <tr>
            <td>Started</td>
            <td style="white-space: nowrap;">{dayjs(run_lifespan.start_ts).format("MMM D, YYYY HH:mm:ss.SSS Z")}</td>
            <td>{run_lifespan.start_comment}</td>
        </tr>
        <tr>
            <td>Completed</td>
            <td style="white-space: nowrap;">{dayjs(run_lifespan.completed_ts).valueOf() > 0 ? dayjs(run_lifespan.completed_ts).format("MMM D, YYYY HH:mm:ss.SSS Z") : "never"}</td>
            <td>{run_lifespan.completed_comment}</td>
        </tr>
        <tr>
            <td>Stopped/Invalidated</td>
            <td style="white-space: nowrap;">{dayjs(run_lifespan.stopped_ts).valueOf() > 0 ? dayjs(run_lifespan.stopped_ts).format("MMM D, YYYY HH:mm:ss.SSS Z") : "never"}</td>
            <td>{run_lifespan.stopped_comment}</td>
        </tr>
        <tr>
            <td>Elapsed</td>
            <td colspan="2"  style="white-space: nowrap;">{calculateElapsed(run_lifespan)}</td>
        </tr>
        <tr>
            <td>Status</td>
            <td colspan="2"  style="white-space: nowrap;">
                <img src={util.runStatusToIconStatic(run_lifespan.final_status)} title={util.runStatusToText(run_lifespan.final_status)} alt=""/>&nbsp;
                {util.runStatusToText(run_lifespan.final_status)}&nbsp;
                {#if run_lifespan.final_status != 3}<button on:click={onStop}>{#if run_lifespan.final_status === 1}Stop{:else}Invalidate{/if}</button>{:else}&nbsp;{/if}</td>
        </tr>
    </tbody>
</table>
{/if}

{#if Object.keys(run_props).length > 0}
    <table>
        <tbody>
            <tr>
                <td>Description:</td>
                <td>{run_props.run_description}</td>
            </tr>
            <tr>
                <td>Script URI:</td>
                <td>{run_props.script_uri}</td>
            </tr>
            <tr>
                <td>Script parameters URI:</td>
                <td>{run_props.script_params_uri}</td>
            </tr>
            <tr>
                <td>Start nodes:</td>
                <td>{run_props.start_nodes}</td>
            </tr>
            <tr>
                <td>Affected nodes:</td>
                <td>{run_props.affected_nodes}</td>
            </tr>
        </tbody>
    </table>
{/if}
