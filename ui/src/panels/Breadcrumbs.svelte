<script>
	const { path_elements = [{ title: 'Empty breadcrumbs' }] } = $props();
</script>

<div>
	<img
		src="i/logo.svg"
		style="width:20px;padding-right:15px;padding-bottom:10px;"
		alt=""
		title="Capillaries-UI"
	/>
	{#each path_elements as e, i}
		{#if !!e.link}
			<a href={e.link}>{e.title}</a>
		{:else}
			{e.title}
		{/if}
		{#if i < path_elements.length - 1}
			&nbsp; &gt; &nbsp;
		{/if}
	{/each}
</div>

<style>
	div {
		display: flex;
	}
</style>
