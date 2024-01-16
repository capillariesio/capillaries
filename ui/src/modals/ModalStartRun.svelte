<script>
	import { onDestroy } from 'svelte';
	import { closeModal } from 'svelte-modals';
	import dayjs from 'dayjs';
	import { webapiUrl, handleResponse } from '../Util.svelte';
	import { ksRunMap } from '../stores.js';

	// provided by Modals
	export let isOpen;

	// Component parameters
	export let keyspace;

	let responseError = '';
	let webapiWaiting = false;
	function setWebapiData(dataFromJson, errorFromJson) {
		webapiWaiting = false;
		if (!!errorFromJson) {
			responseError = "cannot start this run, Capillaries webapi returned an error: " + errorFromJson;
		} else {
			console.log(dataFromJson);
			responseError = '';
			closeModal();
		}
	}

	// Local variables
	let scriptUri = '';
	let paramsUri = '';
	let startNodes = '';
	let runDesc =
		'Manually started from Capillaries-UI at ' + dayjs().format('MMM D, YYYY HH:mm:ss.SSS Z');

	// For this ks, use cached run parameters
	const unsubscribeFromKsRunMap = ksRunMap.subscribe((m) => {
		if (keyspace in m) {
			scriptUri = m[keyspace]['scriptUri'];
			paramsUri = m[keyspace]['paramsUri'];
		}
	});
	onDestroy(unsubscribeFromKsRunMap);

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

	function validateUri(uri) {
		if (uri.trim().length == 0) {
			return 'file URI cannot be empty; ';
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
		//console.log("Sending:",JSON.stringify({"script_uri": scriptUri, "script_params_uri": paramsUri, "start_nodes": startNodes}));
		responseError =
			validateKeyspace(keyspace) +
			validateUri(scriptUri) +
			validateStartNodes(startNodes);
		if (responseError.length == 0) {
			// For this ks, cache last used run parameters
			$ksRunMap[keyspace] = { scriptUri: scriptUri, paramsUri: paramsUri };
			webapiWaiting = true;
			let url = webapiUrl() + '/ks/' + keyspace + '/run';
			let method = 'POST';
			fetch(
				new Request(url, {
					method: method,
					body: JSON.stringify({
						script_uri: scriptUri,
						script_params_uri: paramsUri,
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
			<input bind:value={keyspace} disabled={webapiWaiting} />
			Script URI:
			<input bind:value={scriptUri} disabled={webapiWaiting} />
			Script parameters URI:
			<input bind:value={paramsUri} disabled={webapiWaiting} />
			Start nodes:
			<input bind:value={startNodes} disabled={webapiWaiting} />
			<p style="color:red;">{responseError}</p>
			<div class="actions">
				{#if webapiWaiting}<img
						src="i/wait.svg"
						style="height: 30px;padding-right: 10px;padding-top: 5px;"
						alt=""
					/>{/if}
				<button on:click={closeModal} disabled={webapiWaiting}>Cancel</button>
				<button on:click={newAndCloseModal} disabled={webapiWaiting}>OK</button>
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
