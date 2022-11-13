<script>
    import RunInfo from "../panels/RunInfo.svelte";
    import BatchHistory from "../panels/BatchHistory.svelte";
    export let params; 
    import { onDestroy, onMount } from "svelte";
	import Common, { handleResponse } from '../Common.svelte';
	let common;
    import dayjs from "dayjs";

    import Breadcrumbs from "../panels/Breadcrumbs.svelte";
	let breadcrumbsPathElements = [];

    let webapiData = {run_props:{}, run_lifespan:{}, batch_history: []};

    function setWebapiData(dataFromJson) {
		webapiData = ( !!dataFromJson ? dataFromJson : {run_props:{}, run_lifespan:{}, batch_history: []});
	}

	var timer;

	function fetchData() {
		fetch("http://localhost:6543/ks/" + params.ks_name + "/run/" + params.run_id + "/node/" + params.node_name +"/batch_history")
      		.then(response => response.json())
      		.then(responseJson => { handleResponse(responseJson, setWebapiData);})
      		.catch(error => {console.log(error);});
	}

	onMount(async () => {
        breadcrumbsPathElements = [
            { title:"Keyspaces", link:common.rootLink() },
            { title:params.ks_name + " matrix", link:common.ksMatrixLink(params.ks_name) },
            { title:params.ks_name + "/" + params.run_id + "/" + params.node_name +" batch history" }  ];
    	fetchData();
		timer = setInterval(fetchData, 500);
    });
	onDestroy(async () => {
    	clearInterval(timer);
    });
</script>

<Common bind:this={common} />
<Breadcrumbs bind:pathElements={breadcrumbsPathElements}/>
<RunInfo bind:run_lifespan={webapiData.run_lifespan} bind:run_props={webapiData.run_props}/>
<BatchHistory bind:batch_history={webapiData.batch_history}/>
