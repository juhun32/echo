<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import ScrollArea from '$lib/components/ui/ScrollArea.svelte';
	import { Zap, Database, Clock, RefreshCw } from '@lucide/svelte';

	type HistoryItem = {
		question: string;
		answer: string;
		timestamp: string;
	};

	let history: HistoryItem[] = [];
	let loading = false;
	let interval: any;

	async function fetchHistory() {
		loading = true;
		try {
			const res = await fetch('http://localhost:8080/history');
			if (!res.ok) {
				throw new Error(`History request failed: ${res.status}`);
			}
			const data = await res.json();
			const rows = Array.isArray(data) ? data : [];
			// Map the Go time format to something readable
			history = rows.map((item: any) => ({
				question: item.question || 'Cached Vector Query', // Fallback if empty
				answer: item.answer,
				timestamp: new Date(item.timestamp).toLocaleTimeString([], {
					hour: '2-digit',
					minute: '2-digit'
				})
			}));
		} catch (e) {
			console.error('Failed to fetch history', e);
		} finally {
			loading = false;
		}
	}

	// Auto-refresh every 5 seconds to keep the "Memory" view live
	onMount(() => {
		fetchHistory();
		interval = setInterval(fetchHistory, 5000);
	});

	onDestroy(() => {
		clearInterval(interval);
	});
</script>

<aside class="flex h-full w-[360px] shrink-0 flex-col border-l border-white/5 bg-card">
	<div class="flex items-center justify-between p-4">
		<div>
			<h2 class="flex items-center gap-2 text-xs text-gray-400">Cache History</h2>
			<p class="mt-1 text-[10px] text-gray-600">Local RAM Cache (Active)</p>
		</div>

		<button
			on:click={fetchHistory}
			class="rounded-full p-2 transition-colors hover:bg-white/5"
			title="Refresh Memory"
		>
			<RefreshCw size={14} class={loading ? 'animate-spin text-blue-400' : 'text-gray-500'} />
		</button>
	</div>

	<div class="grid grid-cols-2 gap-px">
		<div class="bg-card text-center">
			<div class="text-xs text-gray-500">Entries</div>
			<div class="text-sm dark:text-white">{history.length}</div>
		</div>
		<div class="bg-card text-center">
			<div class="text-xs text-gray-500">Saved</div>
			<div class="text-sm text-emerald-400">
				{(history.length * 0.5).toFixed(1)}g
			</div>
		</div>
	</div>

	<div class="custom-scrollbar flex-1 space-y-3 overflow-y-auto p-3">
		{#if history.length === 0}
			<div
				class="flex h-full flex-col items-center justify-center space-y-2 text-gray-600 opacity-50"
			>
				<Database size={32} strokeWidth={1} />
				<p class="text-xs">Cache is empty</p>
			</div>
		{/if}

		{#each history as item}
			<div
				class="group relative rounded-lg border border-white/5 bg-card p-3 transition-all hover:border-blue-500/30"
			>
				<div
					class="absolute top-3 right-3 h-1.5 w-1.5 rounded-full bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.6)]"
				></div>

				<h3 class="line-clamp-2 pr-4 text-xs leading-relaxed text-gray-200">
					"{item.question}"
				</h3>

				<div class="mt-2 flex items-center gap-2 font-mono text-[10px] text-gray-500">
					<Clock size={10} />
					<span>{item.timestamp}</span>
				</div>

				<div
					class="mt-2 line-clamp-1 border-t border-white/5 pt-2 text-[10px] text-gray-500 group-hover:text-gray-400"
				>
					↳ {item.answer}
				</div>
			</div>
		{/each}
	</div>

	<div class="border-t border-white/5 bg-card p-2 text-center text-[10px] text-gray-600">
		Vectors stored in-memory
	</div>
</aside>

<style>
	/* Custom Scrollbar */
	.custom-scrollbar::-webkit-scrollbar {
		width: 4px;
	}
	.custom-scrollbar::-webkit-scrollbar-track {
		background: transparent;
	}
	.custom-scrollbar::-webkit-scrollbar-thumb {
		background: #222;
		border-radius: 2px;
	}
	.custom-scrollbar::-webkit-scrollbar-thumb:hover {
		background: #444;
	}
</style>
