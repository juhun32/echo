<script lang="ts">
	import Avatar from '$lib/components/ui/Avatar.svelte';
	import ScrollArea from '$lib/components/ui/ScrollArea.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { ChevronDown } from '@lucide/svelte';

	export let messages: Array<{
		id: string;
		role: 'user' | 'assistant';
		text: string;
		source?: 'CACHE' | 'CLOUD';
		showCard?: boolean;
	}> = [];
	export let loading = false;

	let bottomAnchor: HTMLDivElement;

	function scrollToBottom() {
		bottomAnchor?.scrollIntoView({ behavior: 'smooth', block: 'end' });
	}
</script>

<section class="relative flex min-w-0 flex-1 flex-col bg-background">
	<ScrollArea className="min-h-0 max-h-[calc(100vh-120px)] flex-1 px-8 pt-20 pb-12">
		{#if messages.length === 0}
			<div
				class="mx-auto flex h-full max-w-3xl flex-col items-center justify-center rounded-xl p-6 text-sm text-muted-foreground"
			>
				<p class="rounded-full px-3 py-1 text-xl">Where should we start?</p>
				<p>This is Echo, more energy efficient way to use Gemini web app.</p>
				<p>It uses caching to reduce API calls and carbon footprint.</p>
				<p class="pt-8">Try: "How do I center a div?"</p>
			</div>
		{:else}
			<div class="mx-auto w-full max-w-3xl space-y-6">
				{#each messages as message}
					<div
						class={`flex gap-3 ${message.role === 'user' ? 'justify-end' : 'justify-start pl-8'}`}
					>
						<div
							class={`max-w-[90%] rounded-xl px-4 py-3 text-sm ${message.role === 'user' ? 'bg-card' : 'bg-background'}`}
						>
							{#if message.role === 'assistant' && message.source}
								<div
									class="mb-8 inline-flex rounded-full border border-border px-2.5 py-1 text-[11px] tracking-wide text-muted-foreground"
								>
									{message.source === 'CACHE' ? 'Cache Used: Saved Energy' : 'Gemini API'}
								</div>
							{/if}
							<p class="leading-5 whitespace-pre-wrap">{message.text}</p>
						</div>
					</div>
				{/each}

				{#if loading}
					<div class="text-xs text-muted-foreground">Answering...</div>
				{/if}

				<!-- end anchor -->
				<div bind:this={bottomAnchor}></div>
			</div>
		{/if}
	</ScrollArea>

	{#if messages.length > 0}
		<div class="pointer-events-none absolute right-20 bottom-28 z-[70]">
			<Button
				size="icon"
				variant="outline"
				className="pointer-events-auto h-10 w-10 rounded-full border-border bg-background shadow-md"
				on:click={scrollToBottom}
			>
				<ChevronDown size={18} />
			</Button>
		</div>
	{/if}
</section>
