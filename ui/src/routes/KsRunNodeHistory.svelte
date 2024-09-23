<script>
	import { onDestroy, onMount } from 'svelte';
	import RunInfo from '../panels/RunInfo.svelte';
	import Breadcrumbs from '../panels/Breadcrumbs.svelte';
	import NodeHistory from '../panels/NodeHistory.svelte';
	import Util, { webapiUrl, handleResponse } from '../Util.svelte';
	let util;

	// Route params
	export let params;

	// Breadcrumbs
	let breadcrumbsPathElements = [];

	// Webapi data
	let webapiData = { run_props: {}, run_lifespan: {}, node_history: [] };
	let responseError = '';

	function setWebapiData(dataFromJson, errorFromJson) {
		webapiData = !!dataFromJson
			? dataFromJson
			: { run_props: {}, run_lifespan: {}, node_history: [] };
		if (!!errorFromJson) {
			responseError =
				'cannot retrieve node history, Capillaries webapi returned an error: ' + errorFromJson;
		} else {
			responseError = '';
		}
	}

	var timer;
	let isDestroyed = false;
	function fetchData() {
		let url = webapiUrl() + '/ks/' + params.ks_name + '/run/' + params.run_id + '/node_history';
		let method = 'GET';
		fetch(new Request(url, { method: method }))
			.then((response) => response.json())
			.then((responseJson) => {
				handleResponse(responseJson, setWebapiData);
				if (!isDestroyed) {
					if (webapiData.run_lifespan.final_status > 1) {
						// Run complete, nothing to expect here
						timer = setTimeout(fetchData, 3000);
					} else {
						timer = setTimeout(fetchData, 500);
					}
				}
			})
			.catch((error) => {
				responseError =
					'cannot fetch node history data from Capillaries webapi at ' +
					method +
					' ' +
					url +
					', error:' +
					error;
				console.log(error);
				timer = setTimeout(fetchData, 3000);
			});
	}

	onMount(async () => {
		breadcrumbsPathElements = [
			{ title: 'Keyspaces', link: util.rootLink() },
			{ title: params.ks_name, link: util.ksMatrixLink(params.ks_name) },
			{ title: 'Node history: run ' + params.run_id }
		];
		fetchData();
	});
	onDestroy(async () => {
		isDestroyed = true;
		if (timer) clearTimeout(timer);
	});
</script>

<Util bind:this={util} />
<Breadcrumbs bind:pathElements={breadcrumbsPathElements} />
<p style="color:red;">{responseError}</p>
<RunInfo
	bind:run_lifespan={webapiData.run_lifespan}
	bind:run_props={webapiData.run_props}
	bind:ks_name={params.ks_name}
/>
<NodeHistory bind:node_history={webapiData.node_history} bind:ks_name={params.ks_name} />
