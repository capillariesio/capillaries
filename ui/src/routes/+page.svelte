<script>
	import { page } from '$app/stores';
	import KsMatrix from './KsMatrix.svelte';
	import Keyspaces from './Keyspaces.svelte';
	import KsRunNodeHistory from './KsRunNodeHistory.svelte';
	import KsRunNodeBatchHistory from './KsRunNodeBatchHistory.svelte';
	import { Modals, modals } from 'svelte-modals';
	import { URLPattern } from 'urlpattern-polyfill';

	let routed_page, hg;
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
			hg = patternKsMatrix.exec($page.url).hash.groups;
			routed_page = KsMatrix;
		} else if (patternKsRunNodeHistory.test($page.url)) {
			hg = patternKsRunNodeHistory.exec($page.url).hash.groups;
			routed_page = KsRunNodeHistory;
		} else if (patternKsRunNodeBatchHistory.test($page.url)) {
			hg = patternKsRunNodeBatchHistory.exec($page.url).hash.groups;
			routed_page = KsRunNodeBatchHistory;
		} else if (patternKeyspaces.test($page.url)) {
			routed_page = Keyspaces;
		} else {
			let url = new URL($page.url);
			// Navigate to Keyspaces
			location.href = url.protocol + '//' + url.host + '#/';
		}
	}

	reload();
</script>

<svelte:component
	this={routed_page}
	ks_name={hg.ks_name}
	run_id={hg.run_id}
	node_name={hg.node_name}
/>

<Modals>
	<!-- eslint-disable-next-line -->
	{#snippet backdrop()}
		<div
			slot="backdrop"
			class="backdrop"
			onclick={() => modals.close()}
			onkeypress={() => false}
			aria-hidden="true"
		></div>
	{/snippet}
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
