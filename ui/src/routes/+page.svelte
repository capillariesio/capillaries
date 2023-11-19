<script>
	import { page } from '$app/stores';
	import KsMatrix from './KsMatrix.svelte';
	import Keyspaces from './Keyspaces.svelte';
	import KsRunNodeHistory from './KsRunNodeHistory.svelte';
	import KsRunNodeBatchHistory from './KsRunNodeBatchHistory.svelte';
	import { Modals, closeModal } from 'svelte-modals';
	import { URLPattern } from 'urlpattern-polyfill';

	let params;
	let routed_page;
	const patternKeyspaces = new URLPattern({ hash: '#/' });
	const patternKsMatrix = new URLPattern({ hash: '#/ks/:ks_name/matrix' });
	const patternKsRunNodeBatchHistory = new URLPattern({
		hash: '#/ks/:ks_name/run/:run_id/node/:node_name/batch_history'
	});
	const patternKsRunNodeHistory = new URLPattern({
		hash: '#/ks/:ks_name/run/:run_id/node_history'
	});

	window.addEventListener('hashchange', function () {
		reload();
	});

	function reload() {
		if (patternKsMatrix.test($page.url)) {
			let hg = patternKsMatrix.exec($page.url).hash.groups;
			params = { ks_name: hg.ks_name };
			routed_page = KsMatrix;
		} else if (patternKsRunNodeHistory.test($page.url)) {
			let hg = patternKsRunNodeHistory.exec($page.url).hash.groups;
			params = { ks_name: hg.ks_name, run_id: hg.run_id };
			routed_page = KsRunNodeHistory;
		} else if (patternKsRunNodeBatchHistory.test($page.url)) {
			let hg = patternKsRunNodeBatchHistory.exec($page.url).hash.groups;
			params = { ks_name: hg.ks_name, run_id: hg.run_id, node_name: hg.node_name };
			routed_page = KsRunNodeBatchHistory;
		} else if (patternKeyspaces.test($page.url)) {
			routed_page = Keyspaces;
			params = {};
		} else {
			let url = new URL($page.url);
			// Navigate to Keyspaces
			location.href = url.protocol + '//' + url.host + '#/';
		}
	}

	reload();
</script>

<svelte:component this={routed_page} {params} />

<Modals>
	<div
		slot="backdrop"
		class="backdrop"
		on:click={closeModal}
		on:keypress={() => false}
		aria-hidden="true"
	/>
</Modals>

<style>
	.backdrop {
		position: fixed;
		top: 0;
		bottom: 0;
		right: 0;
		left: 0;
		background: rgba(0, 0, 0, 0.5);
	}
</style>
