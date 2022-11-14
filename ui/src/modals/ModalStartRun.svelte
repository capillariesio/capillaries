<script>
    import { closeModal } from 'svelte-modals'
    import Util, { webapiUrl, handleResponse } from '../Util.svelte';

    // provided by Modals
    export let isOpen

    // Component parameters
    export let keyspace

    let responseError = "";
    function setWebapiData(dataFromJson, errorFromJson) {
		if (!!errorFromJson) {
          responseError = errorFromJson;
        } else {
          console.log(dataFromJson);
          responseError = "";
           closeModal();
        }
    }

    // Local variables
    let scriptUri = "";
    let paramsUri = "";
    let startNodes = "";

    function newAndCloseModal() {
      console.log("Sending:",JSON.stringify({"script_uri": scriptUri, "script_params_uri": paramsUri, "start_nodes": startNodes}));
      fetch(new Request(webapiUrl() + "/ks/" + keyspace + "/run", {method: 'POST', body: JSON.stringify({"script_uri": scriptUri, "script_params_uri": paramsUri, "start_nodes": startNodes})}))
            .then(response => response.json())
            .then(responseJson => { handleResponse(responseJson, setWebapiData);})
            .catch(error => {responseError = error;});
    }
  </script>
  
  {#if isOpen}
  <div role="dialog" class="modal">
    <div class="contents">
      <p>You are about to start a new run in {keyspace}</p>
      Script URI:
      <input bind:value={scriptUri}>
      Script parameters URI:
      <input bind:value={paramsUri}>
      Start nodes:
      <input bind:value={startNodes}>
      <p style="color:red;">{responseError}</p>
      <div class="actions">
        <button on:click="{closeModal}">Cancel</button>
        <button on:click="{newAndCloseModal}">OK</button>
      </div>
    </div>
  </div>
  {/if}
  
  <style>
    .modal {
      position: fixed;
      top: 0;
      bottom: 0;
      right: 0;
      left: 0;
      display: flex;
      justify-content: center;
      align-items: center;
  
      /* allow click-through to backdrop */
      pointer-events: none;
    }
  
    .contents {
      min-width: 80%;
      border-radius: 6px;
      padding: 16px;
      background: white;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
      pointer-events: auto;
    }
  
    .actions {
      margin-top: 32px;
      display: flex;
      justify-content: flex-end;
    }
    button {
  margin: 0px;
	height: 38px;
	padding: 0 30px;
  line-height: 38px;}
  </style>