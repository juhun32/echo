<script lang="ts">
	export let label = 'User';
	export let className = '';

	$: initials = label
		.split(' ')
		.map((part) => part[0])
		.join('')
		.slice(0, 2)
		.toUpperCase();
</script>

<div
	class={`inline-flex h-8 w-8 items-center justify-center rounded-full border border-border bg-muted text-[11px] font-medium text-muted-foreground ${className}`}
	aria-label={label}
>
	{initials}
</div>
