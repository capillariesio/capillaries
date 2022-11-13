<script>
	import Router, {location, link} from 'svelte-spa-router';
	import Keyspaces from './routes/Keyspaces.svelte';
	import KsMatrix from './routes/KsMatrix.svelte';
	import KsRunNodeBatchHistory from './routes/KsRunNodeBatchHistory.svelte';
	import KsRunNodeHistory from './routes/KsRunNodeHistory.svelte';
	import { Modals, closeModal } from 'svelte-modals'
</script>

<Router routes={{
	'/': Keyspaces,
	'/ks/:ks_name/matrix': KsMatrix,
	'/ks/:ks_name/run/:run_id/node/:node_name/batch_history': KsRunNodeBatchHistory,
	'/ks/:ks_name/run/:run_id/node_history': KsRunNodeHistory
}} />

<Modals>
	<div
	  slot="backdrop"
	  class="backdrop"
	  on:click={closeModal}
	  on:keypress={ () => false}
	/>
  </Modals>
  
  <style>
	.backdrop {
	  position: fixed;
	  top: 0;
	  bottom: 0;
	  right: 0;
	  left: 0;
	  background: rgba(0,0,0,0.50)
	}
  </style>
