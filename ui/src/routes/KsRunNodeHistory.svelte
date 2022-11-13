<script>
    import { onDestroy, onMount } from "svelte";
    import RunInfo from "../panels/RunInfo.svelte";
    import Breadcrumbs from "../panels/Breadcrumbs.svelte";
    import NodeHistory from "../panels/NodeHistory.svelte";
	import Util, { webapiUrl, handleResponse } from '../Util.svelte';
	let util;

    // Route params
    export let params; 

	// Breadcrumbs
	let breadcrumbsPathElements = [];

    // Webapi data
    let webapiData = {run_props:{}, run_lifespan:{}, node_history:[]};
    function setWebapiData(dataFromJson) {
		webapiData = ( !!dataFromJson ? dataFromJson : {run_props:{}, run_lifespan:{}, node_history:[]});
        if (webapiData.run_lifespan.final_status > 1) {
            clearInterval(timer);
        }
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
<NodeHistory bind:node_history={webapiData.node_history} bind:ks_name={params.ks_name}/>