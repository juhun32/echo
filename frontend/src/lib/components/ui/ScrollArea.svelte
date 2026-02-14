<script lang="ts">
	export let className = '';
</script>

<div class={`overflow-auto ${className}`}>
	<slot />
</div>
