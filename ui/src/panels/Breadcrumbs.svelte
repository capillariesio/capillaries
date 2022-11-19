<script>
	export let pathElements =[{ title:"Empty breadcrumbs" } ];
</script>

<style>
	div{
		display:flex;
	}
</style>

<div>
	<img src="i/logo.svg" style="width:20px;padding-right:15px;padding-bottom:10px;" alt="" title="Capillaries-UI"/>
	{#each pathElements as e,i}
		{#if !!e.link }
            <a href={e.link}>{e.title}</a>
        {:else}
            {e.title}
        {/if}
		{#if i < pathElements.length - 1}
            &nbsp; &gt; &nbsp;
		{/if}
	{/each}
</div>