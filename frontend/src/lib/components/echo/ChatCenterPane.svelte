<script lang="ts">
	import { marked } from 'marked';
	import DOMPurify from 'isomorphic-dompurify';
	import ScrollArea from '$lib/components/ui/ScrollArea.svelte';

	let {
		messages = [],
		loading = false
	}: {
		messages?: Array<{
			id: string;
			role: 'user' | 'assistant';
			text: string;
			source?: 'CACHE' | 'CLOUD';
			showCard?: boolean;
		}>;
		loading?: boolean;
	} = $props();

	let bottomAnchor = $state<HTMLDivElement | undefined>(undefined);

	marked.setOptions({ breaks: true, gfm: true });

	function renderMarkdown(text: string): string {
		const rawHtml = marked.parse(text);
		return DOMPurify.sanitize(typeof rawHtml === 'string' ? rawHtml : String(rawHtml));
	}

	export function scrollToBottom() {
		bottomAnchor?.scrollIntoView({ behavior: 'smooth', block: 'end' });
	}
</script>

<section class="relative flex min-w-0 flex-1 flex-col bg-background">
	<ScrollArea className="min-h-0 max-h-[calc(100vh-120px)] flex-1 px-8 pt-20">
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
			<div class="mx-auto w-full max-w-3xl space-y-6 pb-20">
				{#each messages as message}
					<div
						class={`flex gap-3 ${message.role === 'user' ? 'justify-end' : 'justify-start pl-8'}`}
					>
						<div
							class={`max-w-[90%] rounded-xl px-4 text-sm ${message.role === 'user' ? 'bg-card' : 'bg-background'}`}
						>
							{#if message.role === 'assistant' && message.source}
								<div
									class="mb-3 inline-flex rounded-full border border-border px-2.5 py-1 text-[11px] tracking-wide text-muted-foreground"
								>
									{message.source === 'CACHE' ? 'Cache Used: Saved Energy' : 'Gemini API'}
								</div>
							{/if}
							<div class="markdown-content leading-5">
								{@html renderMarkdown(message.text)}
							</div>
						</div>
					</div>
				{/each}

				{#if loading}
					<div class="text-xs text-muted-foreground">Answering...</div>
				{/if}
			</div>
		{/if}
		<div bind:this={bottomAnchor}></div>
	</ScrollArea>
</section>

<style>
	.markdown-content :global(p) {
		margin: 0.7rem 0;
		line-height: 1.65;
	}

	.markdown-content :global(h1),
	.markdown-content :global(h2),
	.markdown-content :global(h3),
	.markdown-content :global(h4) {
		margin: 1.25rem 0 0.5rem;
		font-weight: 600;
		line-height: 1.35;
	}

	.markdown-content :global(ul),
	.markdown-content :global(ol) {
		margin: 1rem 0;
		padding-left: 1.35rem;
	}

	.markdown-content :global(ul) {
		list-style-type: disc;
	}

	.markdown-content :global(ol) {
		list-style-type: decimal;
	}

	.markdown-content :global(li) {
		margin: 0.3rem 0;
		line-height: 1.6;
	}

	.markdown-content :global(li > ul),
	.markdown-content :global(li > ol) {
		margin-top: 0.35rem;
	}

	.markdown-content :global(blockquote) {
		margin: 0.9rem 0;
		padding: 0.4rem 0.8rem;
		border-left: 2px solid var(--border);
		color: var(--muted-foreground);
	}

	.markdown-content :global(pre) {
		overflow: auto;
		border: 1px solid var(--border);
		border-radius: 0.5rem;
		padding: 0.75rem;
		margin: 0.75rem 0;
		background: color-mix(in oklab, var(--card) 85%, transparent);
		tab-size: 2;
		white-space: pre;
	}

	.markdown-content :global(code) {
		font-size: 0.85em;
	}

	.markdown-content :global(p code),
	.markdown-content :global(li code) {
		padding: 0.1rem 0.35rem;
		border: 1px solid var(--border);
		border-radius: 0.35rem;
		background: color-mix(in oklab, var(--card) 85%, transparent);
	}
</style>
