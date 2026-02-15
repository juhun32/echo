<script lang="ts">
	import { Cloud, Database, Leaf, RefreshCw } from '@lucide/svelte';
	import type { CacheEntry, CacheStatsResponse, CacheUse } from '$lib/lib/cache';
	import { BACKEND_URL } from '$lib/lib/constants';

	let localRamCache = $state<CacheEntry[]>([]);
	let s3CacheUsed = $state<CacheUse[]>([]);
	let metrics = $state<CacheStatsResponse['metrics']>({
		cacheHits: 0,
		localCacheHits: 0,
		s3CacheHits: 0,
		estimatedTokensSaved: 0,
		energySavedWh: 0,
		co2SavedG: 0
	});
	let constants = $state<CacheStatsResponse['constants']>({
		defaultKWhPer1KTokens: 0.00035,
		gridCO2gPerKWh: 475,
		modelKWhPer1KTokens: {}
	});
	let uploading = $state(false);
	let lastUploadAt = $state('');
	let lastDownloadAt = $state('');
	let loading = $state(false);
	let interval = $state<ReturnType<typeof setInterval> | undefined>(undefined);

	// Maximum words to display for questions/answers
	let { maxWords = 15 }: { maxWords?: number } = $props();

	function truncateWords(text: string, max: number) {
		if (!text) return '';
		const parts = text.trim().split(/\s+/);
		if (parts.length <= max) return text;
		return parts.slice(0, max).join(' ') + '…';
	}

	async function fetchCacheStats() {
		loading = true;
		try {
			const res = await fetch(`${BACKEND_URL}/cache-stats`);
			if (!res.ok) {
				throw new Error(`Cache stats request failed: ${res.status}`);
			}
			const data = (await res.json()) as CacheStatsResponse;

			uploading = data.uploading;
			metrics = data.metrics;
			constants = data.constants ?? constants;
			localRamCache = Array.isArray(data.localRamCache) ? data.localRamCache : [];
			s3CacheUsed = Array.isArray(data.s3CacheUsed) ? data.s3CacheUsed : [];
			lastUploadAt = data.lastUploadAt
				? new Date(data.lastUploadAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
				: '—';
			lastDownloadAt = data.lastDownloadAt
				? new Date(data.lastDownloadAt).toLocaleTimeString([], {
						hour: '2-digit',
						minute: '2-digit'
					})
				: '—';
		} catch (e) {
			console.error('Failed to fetch cache stats', e);
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		fetchCacheStats();
		interval = setInterval(fetchCacheStats, 5000);
		return () => {
			if (interval) {
				clearInterval(interval);
			}
		};
	});
</script>

<aside class="flex h-full w-[85vw] shrink-0 flex-col border-l border-white/5 bg-card lg:w-[360px]">
	<div class="border-b p-4 dark:border-white/5">
		<div class="mb-2 flex items-center justify-between">
			<div>
				<h2 class="flex items-center gap-2 text-xs dark:text-gray-300">
					<Leaf size={12} />
					Environmental Savings
				</h2>
			</div>
			<button
				onclick={() => fetchCacheStats()}
				class="rounded-full p-2 transition-colors hover:bg-white/5"
				title="Refresh Cache Stats"
			>
				<RefreshCw size={14} class={loading ? 'animate-spin text-blue-400' : 'text-gray-500'} />
			</button>
		</div>

		<div class="grid grid-cols-2 gap-2 text-[11px]">
			<div class="rounded-md border p-2 dark:border-white/8">
				<div class="text-gray-500">CO₂ Saved</div>
				<div class="text-emerald-400">{metrics.co2SavedG.toFixed(2)} g</div>
			</div>
			<div class="rounded-md border p-2 dark:border-white/8">
				<div class="text-gray-500">Energy Saved</div>
				<div class="text-emerald-400">{metrics.energySavedWh.toFixed(3)} Wh</div>
			</div>
			<div class="rounded-md border p-2 dark:border-white/8">
				<div class="text-gray-500">Cache Hits</div>
				<div class="dark:text-gray-200">{metrics.cacheHits}</div>
			</div>
			<div class="rounded-md border p-2 dark:border-white/8">
				<div class="text-gray-500">Tokens Saved</div>
				<div class="dark:text-gray-200">{metrics.estimatedTokensSaved}</div>
			</div>
		</div>

		<div class="mt-2 flex items-center justify-between text-[10px] text-gray-500">
			<div class="flex items-center gap-1">
				<Cloud size={12} />
				<span>{uploading ? 'Uploading to S3...' : 'S3 idle'}</span>
			</div>
			<div>Last up: {lastUploadAt}</div>
			<div class="text-[10px] text-gray-600">Last pull: {lastDownloadAt}</div>
		</div>

		<div class="mt-2 text-[10px] text-gray-500">
			<ul class="list-disc pl-4 text-[10px]">
				<li><strong>Tokens saved:</strong> {metrics.estimatedTokensSaved}</li>
				<li><strong>Energy saved (kWh):</strong> {(metrics.energySavedWh / 1000).toFixed(6)}</li>
				<li><strong>CO₂ saved (g):</strong> {metrics.co2SavedG.toFixed(2)}</li>
				<li><strong>Default kWh / 1k tokens:</strong> {constants.defaultKWhPer1KTokens}</li>
			</ul>
		</div>
	</div>

	<div class="min-h-0 flex-1 border-b dark:border-white/5">
		<div class="px-4 pt-3 pb-2 text-xs text-gray-400">Cache Saved</div>
		<div class="h-[calc(100%-34px)] space-y-2 overflow-y-auto px-3 pb-3">
			{#if localRamCache.length === 0}
				<div
					class="flex h-full flex-col items-center justify-center space-y-2 text-gray-600 opacity-60"
				>
					<Database size={28} strokeWidth={1} />
					<p class="text-xs">No local entries yet</p>
				</div>
			{/if}

			{#each localRamCache as item}
				<div class="rounded-lg border bg-card p-3 dark:border-white/6">
					<h3 class="line-clamp-2 text-xs leading-relaxed dark:text-gray-200">
						"{truncateWords(item.question, maxWords)}"
					</h3>
					<div class="mt-2 text-[10px] text-gray-500">↳ {truncateWords(item.answer, maxWords)}</div>
				</div>
			{/each}
		</div>
	</div>

	<div class="min-h-0 flex-1">
		<div class="px-4 pt-3 pb-2 text-xs text-gray-400">S3 Cache Used</div>
		<div class="h-[calc(100%-34px)] space-y-2 overflow-y-auto px-3 pb-3">
			{#if s3CacheUsed.length === 0}
				<div
					class="flex h-full flex-col items-center justify-center space-y-2 text-gray-600 opacity-60"
				>
					<Cloud size={26} strokeWidth={1} />
					<p class="text-xs">No S3 cache hits yet</p>
				</div>
			{/if}

			{#each s3CacheUsed as item}
				<div class="rounded-lg border bg-card p-3 dark:border-white/6">
					<h3 class="line-clamp-2 pr-4 text-xs leading-relaxed dark:text-gray-200">
						"{truncateWords(item.question, maxWords)}"
					</h3>
					<div class="mt-2 text-[10px] text-gray-500">↳ {truncateWords(item.answer, maxWords)}</div>
					<div class="mt-2 flex items-center justify-between text-[10px] text-gray-500">
						<span
							>{new Date(item.timestamp).toLocaleTimeString([], {
								hour: '2-digit',
								minute: '2-digit'
							})}</span
						>
						<span>{item.co2SavedG.toFixed(3)} g CO₂</span>
					</div>
				</div>
			{/each}
		</div>
	</div>
</aside>
