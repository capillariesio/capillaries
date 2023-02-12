<script>
    import { onDestroy, onMount } from "svelte";
	import { openModal } from "svelte-modals";
	import Breadcrumbs from "../panels/Breadcrumbs.svelte";
	import ModalStartRun from "../modals/ModalStartRun.svelte";
	import Util, { webapiUrl, handleResponse } from '../Util.svelte';
	let util;

	// Breadcrumbs
	let breadcrumbsPathElements = [];
    
    // Webapi data
	let webapiData = [];
	let responseError = "";
    function setWebapiData(dataFromJson, errorFromJson) {
		webapiData = ( !!dataFromJson ? dataFromJson : []);
		if (!!errorFromJson) {
			responseError = errorFromJson.msg
		} else {
			responseError = "";
		}
	}

	var timer;
	let isDestroyed = false;
	function fetchData() {
		let url = webapiUrl() + "/ks/";
		let method = "GET";
		fetch(new Request(url, {method: method}))
      		.then(response => response.json())
      		.then(responseJson => {
				handleResponse(responseJson, setWebapiData);
				if (!isDestroyed)
					timer = setTimeout(fetchData, 500);
			})
      		.catch(error => {
				responseError = method + " " + url + ":" + error;
				console.log(error);
				if (!isDestroyed)
					timer = setTimeout(fetchData, 3000);
			});
	}

	onMount(async () => {
		breadcrumbsPathElements = [{ title:"Keyspaces" } ];
    	fetchData();
    });
	onDestroy(async () => {
		isDestroyed = true;
    	if (timer) clearTimeout(timer);
    });

    function onNew() {
        openModal(ModalStartRun, { keyspace: ""});
    }

	let dropResponseError = "";
    function onDrop(keyspace) {
		let url = webapiUrl() + "/ks/" + keyspace;
		let method = "DELETE";
		fetch(new Request(url, {method: method}))
      		.then(response => response.json())
      		.then(responseJson => {
				dropResponseError = (!!responseJson ? responseJson.error.msg : "");
			})
      		.catch(error => {
				dropResponseError = method + " " + url + ":" + error;
				console.log(error);
			});
    }
</script>

<Util bind:this={util} />
<Breadcrumbs bind:pathElements={breadcrumbsPathElements}/>

<p style="color:red;">{responseError}</p>
<p style="color:red;">{dropResponseError}</p>

<button on:click={onNew}>New run</button>
<table>
	<thead>
		<th>Keyspaces ({webapiData.length})</th>
		<th>Drop</th>
	</thead>
	<tbody>
		{#each webapiData as ks}
    		<tr>
	    		<td><a href={util.ksMatrixLink(ks)}>{ks}</a></td>
				<td><button on:click={onDrop(ks)}>Drop</button></td>
	    	</tr>
	    {/each}
	</tbody>
</table>

