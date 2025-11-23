<script>
	import { onDestroy, onMount } from 'svelte';
	import { modals } from 'svelte-modals';
	import Breadcrumbs from '../panels/Breadcrumbs.svelte';
	import ModalStartRun from '../modals/ModalStartRun.svelte';
	import { webapiUrl, handleResponse, ksMatrixLink } from '../Util.svelte';

	let breadcrumbsPathElements = $state([]);

	let webapiData = $state([]);
	let responseError = $state('');
	let dropResponseError = $state('');
	var timer;
	let isDestroyed = false;

	function setWebapiData(dataFromJson, errorFromJson) {
		webapiData = dataFromJson ? dataFromJson : [];
		if (errorFromJson) {
			responseError =
				'cannot retrieve keyspaces, Capillaries webapi returned an error: ' + errorFromJson;
		} else {
			responseError = '';
		}
	}

	function fetchData() {
		let url = webapiUrl() + '/ks/';
		let method = 'GET';
		fetch(new Request(url, { method: method }))
			.then((response) => response.json())
			.then((responseJson) => {
				handleResponse(responseJson, setWebapiData);
				if (!isDestroyed) timer = setTimeout(fetchData, 500);
			})
			.catch((error) => {
				responseError =
					'cannot fetch keyspaces data from Capillaries webapi at ' +
					method +
					' ' +
					url +
					', error:' +
					error;
				console.log(error);
				if (!isDestroyed) timer = setTimeout(fetchData, 3000);
			});
	}

	onMount(() => {
		isDestroyed = false;
		breadcrumbsPathElements = [{ title: 'Keyspaces' }];
		fetchData();
	});

	onDestroy(() => {
		isDestroyed = true;
		if (timer) clearTimeout(timer);
	});

	function onNew() {
		modals.open(ModalStartRun, { keyspace: '' });
	}

	function onDrop(keyspace) {
		let url = webapiUrl() + '/ks/' + keyspace;
		let method = 'DELETE';
		fetch(new Request(url, { method: method }))
			.then((response) => response.json())
			.then((responseJson) => {
				dropResponseError = responseJson ? responseJson.error : '';
			})
			.catch((error) => {
				dropResponseError = method + ' ' + url + ':' + error;
				console.log(error);
			});
	}
</script>

<Breadcrumbs path_elements={breadcrumbsPathElements} />

<p style="color:red;">{responseError}</p>
<p style="color:red;">{dropResponseError}</p>

<button
	onclick={onNew}
	title="Opens a popup to specify parameters (keyspace, script URL etc) for a new run">New run</button
>
<table>
	<thead>
		<tr>
			<th>Keyspaces ({webapiData.length})</th>
			<th>Drop</th>
		</tr>
	</thead>
	<tbody>
		{#each webapiData as ks}
			<tr>
				<td><a href={ksMatrixLink(ks)}>{ks}</a></td>
				<td
					><button title="Drops the keyspace without any warnings" onclick={onDrop(ks)}>Drop</button
					></td
				>
			</tr>
		{/each}
	</tbody>
</table>
