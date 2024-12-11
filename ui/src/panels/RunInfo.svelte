<script>
	import { modals } from 'svelte-modals';
	import dayjs from 'dayjs';
	import ModalStopRun from '../modals/ModalStopRun.svelte';
	import { runStatusToIconStatic, runStatusToText } from '../Util.svelte';

	const { run_lifespan = {}, run_props = {}, ks_name = '' } = $props();

	function onStop() {
		modals.open(ModalStopRun, { ks_name: ks_name, run_id: run_lifespan.run_id });
	}

	function calculateElapsed(ls) {
		if (dayjs(ls.completed_ts).valueOf() > 0) {
			return (dayjs(ls.completed_ts).valueOf() - dayjs(ls.start_ts).valueOf()) / 1000;
		} else if (dayjs(ls.stopped_ts).valueOf() > 0) {
			return (dayjs(ls.stopped_ts).valueOf() - dayjs(ls.start_ts).valueOf()) / 1000;
		} else {
			return (Date.now() - dayjs(ls.start_ts).valueOf()) / 1000;
		}
	}
</script>

{#if Object.keys(run_lifespan).length > 0}
	<table>
		<thead>
			<tr>
				<th colspan="3">Run summary</th>
			</tr>
		</thead>
		<tbody>
			<tr>
				<td>Run Id:</td>
				<td>{run_lifespan.run_id}</td>
				<td></td>
			</tr>
			<tr>
				<td>Started</td>
				<td style="white-space: nowrap;"
					>{dayjs(run_lifespan.start_ts).format('MMM D, YYYY HH:mm:ss.SSS Z')}</td
				>
				<td style="max-width: 1000px; overflow-wrap: break-word;">{run_lifespan.start_comment}</td>
			</tr>
			<tr>
				<td>Completed</td>
				<td style="white-space: nowrap;"
					>{dayjs(run_lifespan.completed_ts).valueOf() > 0
						? dayjs(run_lifespan.completed_ts).format('MMM D, YYYY HH:mm:ss.SSS Z')
						: 'never'}</td
				>
				<td style="max-width: 1000px; overflow-wrap: break-word;"
					>{run_lifespan.completed_comment}</td
				>
			</tr>
			<tr>
				<td>Stopped/Invalidated</td>
				<td style="white-space: nowrap;"
					>{dayjs(run_lifespan.stopped_ts).valueOf() > 0
						? dayjs(run_lifespan.stopped_ts).format('MMM D, YYYY HH:mm:ss.SSS Z')
						: 'never'}</td
				>
				<td style="max-width: 1000px; overflow-wrap: break-word;">{run_lifespan.stopped_comment}</td
				>
			</tr>
			<tr>
				<td>Elapsed</td>
				<td colspan="2" style="white-space: nowrap;">{calculateElapsed(run_lifespan)}</td>
			</tr>
			<tr>
				<td>Status</td>
				<td colspan="2" style="white-space: nowrap;">
					<img
						src={runStatusToIconStatic(run_lifespan.final_status)}
						title={runStatusToText(run_lifespan.final_status)}
						alt=""
					/>&nbsp;
					{runStatusToText(run_lifespan.final_status)}&nbsp;
					{#if run_lifespan.final_status != 3}<button
							onclick={onStop}
							title={run_lifespan.final_status === 1
								? 'Stop run'
								: 'Invalidate the results of a complete run so they cannot be used in depending runs'}
							>{#if run_lifespan.final_status === 1}Stop{:else}Invalidate{/if}</button
						>{:else}&nbsp;{/if}</td
				>
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
				<td>Script URL:</td>
				<td>{run_props.script_url}</td>
			</tr>
			<tr>
				<td>Script parameters URL:</td>
				<td>{run_props.script_params_url}</td>
			</tr>
			<tr>
				<td>Start nodes:</td>
				<td style="max-width: 1000px; overflow-wrap: break-word;">{run_props.start_nodes}</td>
			</tr>
			<tr>
				<td>Affected nodes:</td>
				<td style="max-width: 1000px; overflow-wrap: break-word;">{run_props.affected_nodes}</td>
			</tr>
		</tbody>
	</table>
{/if}

<style>
	tr td:first-child {
		white-space: nowrap;
	}
	img {
		width: 20px;
		vertical-align: text-bottom;
	}
</style>
