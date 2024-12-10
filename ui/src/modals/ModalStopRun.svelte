<script>
	import dayjs from 'dayjs';
	import { webapiUrl, handleResponse } from '../Util.svelte';

	const { isOpen, close, ks_name, run_id } = $props();

	let responseError = $state('');
	let webapiWaiting = $state(false);
	let stopComment = $state(
		'Manually stopped/invalidated from Capillaries-UI at ' +
			dayjs().format('MMM D, YYYY HH:mm:ss.SSS Z')
	);

	function setWebapiData(dataFromJson, errorFromJson) {
		webapiWaiting = false;
		if (errorFromJson) {
			responseError =
				'cannot stop this run, Capillaries webapi returned an error: ' + errorFromJson;
		} else {
			responseError = '';
			close();
		}
	}

	function stopAndCloseModal() {
		webapiWaiting = true;
		let url = webapiUrl() + '/ks/' + ks_name + '/run/' + run_id;
		let method = 'DELETE';
		fetch(new Request(url, { method: method, body: '{"comment": "' + stopComment + '"}' }))
			.then((response) => response.json())
			.then((responseJson) => {
				handleResponse(responseJson, setWebapiData);
			})
			.catch((error) => {
				webapiWaiting = false;
				responseError = method + ' ' + url + ':' + error;
			});
	}
</script>

{#if isOpen}
	<div role="dialog" class="modal">
		<div class="contents">
			<p>You are about to stop/invalidate run {run_id} in {ks_name}</p>
			Comment (will be stored in run history):
			<input value={stopComment} disabled={webapiWaiting} />
			<p style="color:red;">{responseError}</p>

			<div class="actions">
				{#if webapiWaiting}<img
						src="i/wait.svg"
						style="height: 30px;padding-right: 10px;padding-top: 5px;"
						alt=""
					/>{/if}
				<button onclick={close} disabled={webapiWaiting}>Cancel</button>
				<button onclick={stopAndCloseModal} disabled={webapiWaiting}>OK</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.modal {
		position: fixed;
		top: 0;
		bottom: 0;
		right: 0;
		left: 0;
		display: flex;
		justify-content: center;
		align-items: center;

		/* allow click-through to backdrop */
		pointer-events: none;
	}

	.contents {
		min-width: 80%;
		border-radius: 6px;
		padding: 16px;
		background: white;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		pointer-events: auto;
	}

	.actions {
		margin-top: 32px;
		display: flex;
		justify-content: flex-end;
	}
	button {
		margin: 0px;
		height: 38px;
		padding: 0 30px;
		line-height: 38px;
	}
</style>
