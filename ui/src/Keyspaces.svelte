<script>
    import { onDestroy, onMount } from "svelte";

	import { handleResponse } from './Common.svelte';

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
    	fetchData();
		timer = setInterval(fetchData, 500);
    });
	onDestroy(async () => {
    	clearInterval(timer);
    });
</script>

<h1>All keyspaces: {webapiData.length}</h1>

<table>
	<thead>
		<th>Keyspace name</th>
	</thead>
	<tbody>
		{#each webapiData as ks}
    		<tr>
	    		<td><a href="/#/ks/{ks}/matrix">{ks}</a></td>
	    	</tr>
	    {/each}
	</tbody>
</table>
