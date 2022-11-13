<script>
    import { onDestroy, onMount } from "svelte";
    import dayjs from "dayjs";
    import RunInfo from "../panels/RunInfo.svelte";
    import Breadcrumbs from "../panels/Breadcrumbs.svelte";
	import Util, { webapiUrl, handleResponse } from '../Util.svelte';
	let util;

    // Route params
    export let params; 

	// Breadcrumbs
	let breadcrumbsPathElements = [];

    // Webapi data
    let webapiData = {run_props:{}, run_lifespan:{}};
    function setWebapiData(dataFromJson) {
		webapiData = ( !!dataFromJson ? dataFromJson : {run_props:{}, run_lifespan:{}});
	}

	function fetchData() {
		fetch(webapiUrl() + "/ks/" + params.ks_name + "/run/" + params.run_id + "/node_history")
      		.then(response => response.json())
      		.then(responseJson => { handleResponse(responseJson, setWebapiData);})
      		.catch(error => {console.log(error);});
	}

	var timer;
	onMount(async () => {
        breadcrumbsPathElements = [
            { title:"Keyspaces", link:util.rootLink() },
            { title:params.ks_name + " matrix", link:util.ksMatrixLink(params.ks_name) },
            { title:params.ks_name + "/" + params.run_id + " node history" }  ];
    	fetchData();
		timer = setInterval(fetchData, 500);
    });
	onDestroy(async () => {
    	clearInterval(timer);
    });
</script>

<Util bind:this={util} />
<Breadcrumbs bind:pathElements={breadcrumbsPathElements}/>
<RunInfo bind:run_lifespan={webapiData.run_lifespan} bind:run_props={webapiData.run_props}/>
