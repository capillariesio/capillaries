<script>
    import { closeModal } from 'svelte-modals'
    import Util, { handleResponse } from '../Util.svelte';

    // provided by Modals
    export let isOpen

    // Component parameters
    export let run_id
    export let keyspace

    let responseError = "";
    function setWebapiData(dataFromJson, errorFromJson) {
		if (!!errorFromJson) {
            responseError = errorFromJson;
        } else {
            responseError = "";
            closeModal();
        }
    }

    let stopComment = "Stopped using capillaries-ui";

    function stopAndCloseModal() {
		fetch(new Request(Util.webapiUrl + "/ks/" + keyspace + "/run/" + run_id, {method: 'DELETE', body: '{"comment": "' + stopComment +'"}'}))
      		.then(response => response.json())
      		.then(responseJson => { handleResponse(responseJson, setWebapiData);})
      		.catch(error => {console.log(error);});
        
    }
  </script>
  
  {#if isOpen}
  <div role="dialog" class="modal">
    <div class="contents">
      <h2>Stop {keyspace}/{run_id}?</h2>
      <h4>Comment:</h4>
      <input value={stopComment}>
      <h4>{responseError}</h4>

      <div class="actions">
        <button on:click="{closeModal}">Cancel</button>
        <button on:click="{stopAndCloseModal}">OK</button>
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
      min-width: 240px;
      border-radius: 6px;
      padding: 16px;
      background: white;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
      pointer-events: auto;
    }
  
    h2 {
      text-align: center;
      font-size: 24px;
    }
  
 
    .actions {
      margin-top: 32px;
      display: flex;
      justify-content: flex-end;
    }
  </style>