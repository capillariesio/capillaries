<script>
	import Router, {location, link} from 'svelte-spa-router';
	import Keyspaces from './Keyspaces.svelte';
	import KsMatrix from './KsMatrix.svelte';
	import KsRunNodeBatchHistory from './KsRunNodeBatchHistory.svelte';
</script>

<Router routes={{
	'/': Keyspaces,
	'/ks/:ks_name/matrix': KsMatrix,
	'/ks/:ks_name/run/:run_id/node/:node_name/batch_history': KsRunNodeBatchHistory
}} />
