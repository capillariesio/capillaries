<script>
	const {items = []} = $props();
	let activeTabValue = $state(1);
  
	const handleClick = tabValue => () => (activeTabValue = tabValue);
  </script>
  
  <ul>
  {#each items as item}
	  <li class={activeTabValue === item.value ? 'active' : ''}>
		  <button onclick={handleClick(item.value)}>{item.label}</button>

	  </li>
  {/each}
  </ul>
  {#each items as item}
	  {#if activeTabValue == item.value}
	  <div class="box">
		  {@render item.component()}
	  </div>
	  {/if}
  {/each}
  <style>
	  .box {
			width:100%;
		  margin-bottom: 10px;
		  padding-top: 10px;
		  display: inline-block;
	  }
	ul {
	  display: flex;
	  flex-wrap: wrap;
	  padding-left: 0;
	  margin-bottom: 0;
	  list-style: none;
	  border-bottom: 1px solid #dee2e6;
	}
	  li {
		  margin-bottom: -1px;
	  }
  
	button {
	  border: 1px solid transparent;
	  border-top-left-radius: 0.25rem;
	  border-top-right-radius: 0.25rem;
	  border-bottom-left-radius: 0;
	  border-bottom-right-radius: 0;
	  display: block;
	  padding: 0.5rem 1rem;
	  cursor: pointer;
	}
  
	button:hover {
	  border-color: #e9ecef #e9ecef #dee2e6;
	}

	button:focus{
		text-decoration: underline;
	}
  
	li.active > button {
	  color: #495057;
	  background-color: #fff;
	  border-color: #dee2e6 #dee2e6 #fff;
	}
  </style>