<script lang="ts">
	export let variant: 'default' | 'outline' | 'ghost' = 'default';
	export let size: 'sm' | 'md' | 'icon' = 'md';
	export let type: 'button' | 'submit' | 'reset' = 'button';
	export let disabled = false;
	export let className = '';

	const base =
		'inline-flex items-center justify-center rounded-full text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50';

	$: variantClass =
		variant === 'outline'
			? 'border border-border bg-transparent hover:bg-muted'
			: variant === 'ghost'
				? 'hover:bg-muted'
				: 'bg-foreground text-background hover:opacity-90';

	$: sizeClass = size === 'sm' ? 'h-8 px-3' : size === 'icon' ? 'h-9 w-9' : 'h-9 px-4 py-2';
</script>

<button {type} {disabled} class={`${base} ${variantClass} ${sizeClass} ${className}`} on:click>
	<slot />
</button>
