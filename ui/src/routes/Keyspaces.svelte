<script>
    import { onDestroy, onMount } from "svelte";
	import Breadcrumbs from "../panels/Breadcrumbs.svelte";

	let breadcrumbsPathElements = [];

	import Common , { handleResponse } from '../Common.svelte';
	let common;

    let webapiData = [];

    function setWebapiData(dataFromJson) {
		webapiData = ( !!dataFromJson ? dataFromJson : []);
	}

	var timer;

	function fetchData() {
		fetch("http://localhost:6543/ks/")
      		.then(response => response.json())
      		.then(responseJson => { handleResponse(responseJson, setWebapiData);})
      		.catch(error => {console.log(error);});
	}

	onMount(async () => {
		breadcrumbsPathElements = [{ title:"Keyspaces" } ];
    	fetchData();
		timer = setInterval(fetchData, 500);
    });
	onDestroy(async () => {
    	clearInterval(timer);
    });
</script>


<Breadcrumbs bind:pathElements={breadcrumbsPathElements}/>
<Common bind:this={common} />

<table>
	<thead>
		<th>Keyspaces ({webapiData.length})</th>
	</thead>
	<tbody>
		{#each webapiData as ks}
    		<tr>
	    		<td><a href={common.ksMatrixLink(ks)}>{ks}</a></td>
	    	</tr>
	    {/each}
	</tbody>
</table>
