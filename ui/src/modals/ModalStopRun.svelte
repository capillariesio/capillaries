<script>
    import { closeModal } from 'svelte-modals'
    import dayjs from "dayjs";
    import Util, { webapiUrl, handleResponse } from '../Util.svelte';

    // provided by Modals
    export let isOpen

    // Component parameters
    export let run_id
    export let keyspace

    let responseError = "";
    function setWebapiData(dataFromJson, errorFromJson) {
		if (!!errorFromJson) {
            responseError = errorFromJson.error.msg;
        } else {
            responseError = "";
            closeModal();
        }
    }

    // Local variables
    let stopComment = "Stopped using capillaries-ui at " +  dayjs().format("MMM D, YYYY HH:mm:ss.SSS Z");

    function stopAndCloseModal() {
		fetch(new Request(webapiUrl() + "/ks/" + keyspace + "/run/" + run_id, {method: 'DELETE', body: '{"comment": "' + stopComment +'"}'}))
      		.then(response => response.json())
      		.then(responseJson => { handleResponse(responseJson, setWebapiData);})
      		.catch(error => {responseError = error;});
        
    }
  </script>
  
  {#if isOpen}
  <div role="dialog" class="modal">
    <div class="contents">
      <p>You are about to stop run {run_id} in {keyspace}</p>
      Comment (will be stored in run history):
      <input bind:value={stopComment}>
      <p style="color:red;">{responseError}</p>

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