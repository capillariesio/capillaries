<script>
  import { openModal } from 'svelte-modals'
  import ModalStopRun from '../modals/ModalStopRun.svelte'
  import dayjs from "dayjs";
  import Common from '../Common.svelte'; let common;

  export let run_lifespan = {};
  export let run_props = {};
  export let keyspace = "";
  export let run_id = 0;

  function onStop() {
    openModal(ModalStopRun, { keyspace: keyspace, run_id: run_id})
  }
    
    function calculateElapsed(ls) {
        if (dayjs(ls.completed_ts).valueOf() > 0) {
            return (dayjs(ls.completed_ts).valueOf()-dayjs(ls.start_ts).valueOf())/1000;
        } else if (dayjs(ls.stopped_ts).valueOf() > 0) {
            return (dayjs(ls.stopped_ts).valueOf()-dayjs(ls.start_ts).valueOf())/1000;
        } else {
            return (Date.now()-dayjs(ls.start_ts).valueOf())/1000;
        }
    }
</script>

<Common bind:this={common} />

{#if Object.keys(run_lifespan).length > 0}
<table>
    <thead>
        <th>Run id</th>
        <th>Started</th>
        <th>Completed</th>
        <th>Stopped</th>
        <th>Elapsed</th>
        <th>Status</th>
        <th>{#if run_lifespan.final_status === 1}Stop{/if}</th>
    </thead>
	<tbody>
        <tr>
            <td>{run_lifespan.run_id}</td>
            <td>{dayjs(run_lifespan.start_ts).format("MMM D, YYYY HH:mm:ss.SSS Z")}</td>
            <td>{(dayjs(run_lifespan.completed_ts).valueOf() > 0 ? dayjs(run_lifespan.completed_ts).format("MMM D, YYYY HH:mm:ss.SSS Z") : "never")}</td>
            <td>{(dayjs(run_lifespan.stopped_ts).valueOf() > 0 ? dayjs(run_lifespan.stopped_ts).format("MMM D, YYYY HH:mm:ss.SSS Z") : "never")}</td>
            <td>{calculateElapsed(run_lifespan)}</td>
            <td><img src={common.runStatusToIcon(run_lifespan.final_status)} title={common.runStatusToText(run_lifespan.final_status)} alt=""/></td>
            <td>
                {#if run_lifespan.final_status === 1}
                    <button on:click="{onStop}">Stop</button>
                {:else}
                    &nbsp;
                {/if}
            </td>
        </tr>
    </tbody>
</table>
{/if}

{#if Object.keys(run_props).length > 0}
<table>
	<tbody>
        <tr>
            <td style="white-space: nowrap;">Start nodes:</td>
            <td>{run_props.start_nodes}</td>
        </tr>
        <tr>
            <td style="white-space: nowrap;">Affected nodes:</td>
            <td>{run_props.affected_nodes}</td>
        </tr>
        <tr>
            <td style="white-space: nowrap;">Script URI:</td>
            <td>{run_props.script_uri}</td>
        </tr>
        <tr>
            <td style="white-space: nowrap;">Script parameters URI:</td>
            <td>{run_props.script_params_uri}</td>
        </tr>
    </tbody>
</table>
{/if}
