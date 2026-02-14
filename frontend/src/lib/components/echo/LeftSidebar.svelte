<script lang="ts">
	import Button from '$lib/components/ui/Button.svelte';
	import ScrollArea from '$lib/components/ui/ScrollArea.svelte';
	import Separator from '$lib/components/ui/Separator.svelte';
	import { Moon, Plus, Sun } from '@lucide/svelte';

	export let recentChats: Array<{ id: string; title: string }> = [];
	export let activeChatId = '';
	export let isDark = true;
	export let onNewChat: () => void = () => {};
	export let onSelectChat: (id: string) => void = () => {};
	export let onToggleTheme: () => void = () => {};
</script>

<aside class="flex h-full w-[260px] shrink-0 flex-col bg-card dark:text-[#E3E3E3]">
	<div class="p-4">
		<!-- <div class="text mb-3 font-medium tracking-tight">Echo (Demo)</div> -->
		<div class="flex items-center justify-between gap-3">
			<div>
				<Button className="h-8 w-8" variant="outline" size="icon" on:click={onToggleTheme}>
					{#if isDark}
						<Sun size={16} />
					{:else}
						<Moon size={16} />
					{/if}
				</Button>
			</div>
			<Button className="w-full justify-end gap-1" size="sm" on:click={onNewChat}
				>New Chat <Plus size={16} /></Button
			>
		</div>
	</div>

	<Separator />

	<div class="px-4 pt-3 pb-2 text-xs tracking-wide text-gray-500">Chats</div>
	<ScrollArea className="min-h-0 flex-1 px-2 pb-3">
		<div class="space-y-1">
			{#each recentChats as chat}
				<Button
					variant={activeChatId === chat.id ? 'outline' : 'ghost'}
					className="w-full justify-start truncate"
					on:click={() => onSelectChat(chat.id)}
				>
					{chat.title}
				</Button>
			{/each}
		</div>
	</ScrollArea>

	<Separator />

	<div class="flex gap-3 p-4">
		<div class="text-xs tracking-wide text-gray-500">Tip</div>
		<div class="bg-card text-xs">
			Echo uses caching to save energy. Try asking the same question twice and see the difference!
		</div>
	</div>
</aside>
