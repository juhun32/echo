<script lang="ts">
	export let orientation: 'horizontal' | 'vertical' = 'horizontal';
	export let className = '';
</script>

{#if orientation === 'vertical'}
	<div
		class={`h-full w-px shrink-0 bg-border ${className}`}
		role="separator"
		aria-orientation="vertical"
	></div>
{:else}
	<div
		class={`h-px w-full shrink-0 bg-border ${className}`}
		role="separator"
		aria-orientation="horizontal"
	></div>
{/if}
