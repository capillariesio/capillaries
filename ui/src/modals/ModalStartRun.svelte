<script>
	import { onDestroy } from 'svelte';
	import dayjs from 'dayjs';
	import { webapiUrl, handleResponse } from '../Util.svelte';
	import { ksRunMap } from '../stores.js';

	let { isOpen, close, ks_name } = $props();

	let responseError = $state('');
	let webapiWaiting = $state(false);
	let scriptUrl = $state('');
	let paramsUrl = $state('');
	let startNodes = $state('');
	let runDesc = $state(
		'Manually started from Capillaries-UI at ' + dayjs().format('MMM D, YYYY HH:mm:ss.SSS Z')
	);

	function setWebapiData(dataFromJson, errorFromJson) {
		webapiWaiting = false;
		if (errorFromJson) {
			responseError =
				'cannot start this run, Capillaries webapi returned an error: ' + errorFromJson;
		} else {
			console.log(dataFromJson);
			responseError = '';
			close();
		}
	}

	// For this ks, use cached run parameters
	const ksRunMapUnsubscriberFunc = ksRunMap.subscribe((m) => {
		if (ks_name in m) {
			scriptUrl = m[ks_name]['scriptUrl'];
			paramsUrl = m[ks_name]['paramsUrl'];
			startNodes = m[ks_name]['startNodes'];
		}
	});

	onDestroy(ksRunMapUnsubscriberFunc);

	function validateKeyspace(ks) {
		if (ks.trim().length == 0) {
			return 'keyspace cannot be empty; ';
		}
		if (ks.startsWith('system')) {
			return "keyspace name cannot start with 'system'; ";
		}
		const allowedPattern = /[a-zA-Z0-9_]+/g;
		if (!ks.match(allowedPattern)) {
			return 'keyspace name allowed pattern: ' + allowedPattern + '; ';
		}
		return '';
	}

	function validateUrl(url) {
		if (url.trim().length == 0) {
			return 'file URL cannot be empty; ';
		}
		return '';
	}

	function validateStartNodes(sn) {
		if (sn.trim().length == 0) {
			return 'start nodes string cannot be empty; ';
		}
		const allowedPattern = /[a-zA-Z0-9_,]+/g;
		if (!sn.match(allowedPattern)) {
			return 'start node list must be comma-separated, only a-zA-Z0-9_ allowed in node names; ';
		}
		return '';
	}

	function newAndCloseModal() {
		responseError =
			validateKeyspace(ks_name) + validateUrl(scriptUrl) + validateStartNodes(startNodes);
		if (responseError.length == 0) {
			// For this ks, cache last used run parameters
			$ksRunMap[ks_name] = { scriptUrl: scriptUrl, paramsUrl: paramsUrl, startNodes: startNodes };
			webapiWaiting = true;
			let url = webapiUrl() + '/ks/' + ks_name + '/run';
			let method = 'POST';
			fetch(
				new Request(url, {
					method: method,
					body: JSON.stringify({
						script_url: scriptUrl,
						script_params_url: paramsUrl,
						start_nodes: startNodes,
						run_description: runDesc
					})
				})
			)
				.then((response) => response.json())
				.then((responseJson) => {
					handleResponse(responseJson, setWebapiData);
				})
				.catch((error) => {
					webapiWaiting = false;
					responseError = method + ' ' + url + ':' + error;
				});
		}
	}
</script>

{#if isOpen}
	<div role="dialog" class="modal">
		<div class="contents">
			<p>You are about to start a new run</p>
			Run description:
			<input bind:value={runDesc} disabled={webapiWaiting} />
			Keyspace:
			<input bind:value={ks_name} disabled={webapiWaiting} />
			Script URL:
			<input bind:value={scriptUrl} disabled={webapiWaiting} />
			Script parameters URL:
			<input bind:value={paramsUrl} disabled={webapiWaiting} />
			Start nodes:
			<input bind:value={startNodes} disabled={webapiWaiting} />
			<p style="color:red;">{responseError}</p>
			<div class="actions">
				{#if webapiWaiting}<img
						src="i/wait.svg"
						style="height: 30px;padding-right: 10px;padding-top: 5px;"
						alt=""
					/>{/if}
				<button onclick={close} disabled={webapiWaiting}>Cancel</button>
				<button onclick={newAndCloseModal} disabled={webapiWaiting}>OK</button>
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
