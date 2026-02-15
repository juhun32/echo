<script lang="ts">
	import Button from '$lib/components/ui/Button.svelte';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import ScrollArea from '$lib/components/ui/ScrollArea.svelte';
	import Separator from '$lib/components/ui/Separator.svelte';
	import {
		Ellipsis,
		FolderPlus,
		Lightbulb,
		Moon,
		Pencil,
		Pin,
		PinOff,
		Plus,
		Settings,
		Sun,
		Trash2
	} from '@lucide/svelte';

	let {
		recentChats = [],
		folders = [],
		activeChatId = '',
		isDark = true,
		onNewChat = () => {},
		onSelectChat = () => {},
		onToggleTheme = () => {},
		onRenameChat = () => {},
		onDeleteChat = () => {},
		onTogglePinChat = () => {},
		onMoveChatToFolder = () => {},
		onCreateFolder = () => {}
	}: {
		recentChats?: Array<{ id: string; title: string; pinned?: boolean; folder?: string }>;
		folders?: string[];
		activeChatId?: string;
		isDark?: boolean;
		onNewChat?: () => void;
		onSelectChat?: (id: string) => void;
		onToggleTheme?: () => void;
		onRenameChat?: (id: string) => void;
		onDeleteChat?: (id: string) => void;
		onTogglePinChat?: (id: string) => void;
		onMoveChatToFolder?: (id: string) => void;
		onCreateFolder?: () => void;
	} = $props();

	let tipDismissed = $state(false);

	let folderGroups = $derived(
		folders
			.map((folder) => ({
				folder,
				chats: recentChats.filter((chat) => (chat.folder ?? 'General') === folder)
			}))
			.filter((group) => group.chats.length > 0)
	);
</script>

<aside class="flex h-full w-[260px] shrink-0 flex-col bg-card dark:text-[#E3E3E3]">
	<div class="px-3 py-4">
		<div class="mb-2 flex items-center justify-between">
			<Button className="h-8 w-8 rounded-full" variant="outline" size="icon">
				<Settings size={16} />
				<span class="sr-only">Settings</span>
			</Button>
			<div class="flex items-baseline justify-end gap-1 px-3 tracking-tight">
				<p class="text-xs">Welcome,</p>
				User
			</div>
		</div>
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

	<div
		class="flex items-center justify-between px-4 pt-3 pb-2 text-xs tracking-wide text-muted-foreground"
	>
		<span>Chats</span>
		<Button variant="ghost" size="icon" className="h-7 w-7" on:click={onCreateFolder}>
			<FolderPlus size={14} />
			<span class="sr-only">Create folder</span>
		</Button>
	</div>
	<ScrollArea className="min-h-0 flex-1 px-3 pb-3">
		{#if recentChats.length === 0}
			<div class="rounded-lg border border-dashed border-border p-3 text-xs text-muted-foreground">
				No chats yet. Create one to get started.
			</div>
		{:else}
			<div class="space-y-3">
				{#each folderGroups as group (group.folder)}
					<div>
						<p class="mb-1 px-1 text-[11px] font-medium tracking-wide text-muted-foreground">
							{group.folder}
						</p>
						<div class="space-y-1">
							{#each group.chats as chat (chat.id)}
								<div class="group flex items-center gap-1">
									<Button
										variant={activeChatId === chat.id ? 'outline' : 'ghost'}
										className="min-w-0 flex-1 justify-start truncate gap-1"
										on:click={() => onSelectChat(chat.id)}
									>
										{#if chat.pinned}
											<Pin size={12} />
										{/if}
										<p class="max-w-[150px] truncate" title={chat.title}>{chat.title}</p>
									</Button>
									<DropdownMenu.Root>
										<DropdownMenu.Trigger
											class="inline-flex h-7 w-7 items-center justify-center rounded-md opacity-0 transition-opacity group-hover:opacity-100 hover:bg-muted data-[state=open]:opacity-100"
										>
											<Ellipsis size={14} />
											<span class="sr-only">Manage chat</span>
										</DropdownMenu.Trigger>
										<DropdownMenu.Content align="end" sideOffset={6}>
											<DropdownMenu.Item onSelect={() => onTogglePinChat(chat.id)}>
												{#if chat.pinned}
													<PinOff size={12} />
													Unpin chat
												{:else}
													<Pin size={12} />
													Pin chat
												{/if}
											</DropdownMenu.Item>
											<DropdownMenu.Item onSelect={() => onMoveChatToFolder(chat.id)}>
												<FolderPlus size={12} />
												Move to folder
											</DropdownMenu.Item>
											<DropdownMenu.Item onSelect={() => onRenameChat(chat.id)}>
												<Pencil size={12} />
												Rename chat
											</DropdownMenu.Item>
											<DropdownMenu.Separator />
											<DropdownMenu.Item
												variant="destructive"
												onSelect={() => onDeleteChat(chat.id)}
											>
												<Trash2 size={12} />
												Delete chat
											</DropdownMenu.Item>
										</DropdownMenu.Content>
									</DropdownMenu.Root>
								</div>
							{/each}
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</ScrollArea>

	<Separator />

	<div class="flex gap-2 p-3">
		<div class="text-xs tracking-wide text-gray-500"><Lightbulb size={14} /></div>
		<div class="bg-card text-xs">
			Echo uses caching to save energy. Try asking the same question twice and see the difference!
		</div>
	</div>
</aside>
