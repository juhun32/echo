<script lang="ts">
	import Button from './Button.svelte';

	export let items: Array<{ value: string; label: string }> = [];
	export let value = '';
	export let className = '';

	const select = (next: string) => {
		value = next;
	};
</script>

<div class={`flex h-full flex-col ${className}`}>
	<div class="inline-flex w-fit items-center rounded-md border border-border bg-muted/40 p-1">
		{#each items as item}
			<Button
				variant={value === item.value ? 'default' : 'ghost'}
				size="sm"
				className={`h-7 px-3 text-xs ${value === item.value ? '' : 'text-muted-foreground'}`}
				on:click={() => select(item.value)}
			>
				{item.label}
			</Button>
		{/each}
	</div>
	<div class="mt-3 min-h-0 flex-1">
		<slot {value} />
	</div>
</div>
