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

	function fetchData() {
		fetch(webapiUrl() + "/ks/")
      		.then(response => response.json())
      		.then(responseJson => { handleResponse(responseJson, setWebapiData);})
      		.catch(error => {responseError = error;console.log(error)});
	}

	var timer;
	onMount(async () => {
		breadcrumbsPathElements = [{ title:"Keyspaces" } ];
    	fetchData();
		timer = setInterval(fetchData, 500);
    });
	onDestroy(async () => {
    	clearInterval(timer);
    });

    function onNew() {
        openModal(ModalStartRun, { keyspace: ""});
    }

	let dropResponseError = "";
    function onDrop(keyspace) {
		fetch(new Request(webapiUrl() + "/ks/" + keyspace, {method: 'DELETE'}))
      		.then(response => response.json())
      		.then(responseJson => { dropResponseError = (!!responseJson ? responseJson.error.msg : ""); })
      		.catch(error => {dropResponseError = error;console.log(error);});
    }
</script>

<Util bind:this={util} />
<Breadcrumbs bind:pathElements={breadcrumbsPathElements}/>
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

<p style="color:red;">{responseError}</p>
<p style="color:red;">{dropResponseError}</p>