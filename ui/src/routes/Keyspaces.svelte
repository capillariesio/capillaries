<script>
    import { onDestroy, onMount } from "svelte";
	import Breadcrumbs from "../panels/Breadcrumbs.svelte";
	import Util, { webapiUrl, handleResponse } from '../Util.svelte';
	let util;

	// Breadcrumbs
	let breadcrumbsPathElements = [];
    
    // Webapi data
	let webapiData = [];
    function setWebapiData(dataFromJson) {
		webapiData = ( !!dataFromJson ? dataFromJson : []);
	}

	function fetchData() {
		fetch(webapiUrl() + "/ks/")
      		.then(response => response.json())
      		.then(responseJson => { handleResponse(responseJson, setWebapiData);})
      		.catch(error => {console.log(error);});
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
</script>

<Util bind:this={util} />
<Breadcrumbs bind:pathElements={breadcrumbsPathElements}/>

<table>
	<thead>
		<th>Keyspaces ({webapiData.length})</th>
	</thead>
	<tbody>
		{#each webapiData as ks}
    		<tr>
	    		<td><a href={util.ksMatrixLink(ks)}>{ks}</a></td>
	    	</tr>
	    {/each}
	</tbody>
</table>
