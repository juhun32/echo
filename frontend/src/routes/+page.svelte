<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { ChevronLeft, ChevronRight } from '@lucide/svelte';
	import LeftSidebar from '$lib/components/echo/LeftSidebar.svelte';
	import ChatCenterPane from '$lib/components/echo/ChatCenterPane.svelte';
	import RightWorkspacePane from '$lib/components/echo/RightWorkspacePane.svelte';
	import FloatingInputBar from '$lib/components/echo/FloatingInputBar.svelte';

	// ui
	import Separator from '$lib/components/ui/Separator.svelte';

	// lib
	import type { Message, ChatApiResponse, FeatureExtractor, ChatSession } from '$lib/lib/chat';
	import { MODEL_NAME } from '$lib/lib/chat';
	import { BACKEND_URL } from '$lib/lib/constants';

	// demo import
	import { chatSessions as importedChatSessions } from '$lib/lib/demo';

	let input = $state('');
	let loading = $state(false);
	let isDark = $state(true);
	let selectedModel = $state('gemini-2.5-flash-lite');
	let leftSidebarCollapsed = $state(false);
	let rightSidebarCollapsed = $state(false);
	let rightPaneWidth = $state(360);
	let isMobile = $state(false);
	let inputFocusTrigger = $state(0);
	let rightResizeCleanup: (() => void) | null = null;
	let chatCenterPaneRef = $state<{
		scrollToBottom?: () => void;
	} | null>(null);

	const SIDEBAR_LEFT_STORAGE_KEY = 'echo:left-sidebar-collapsed';
	const SIDEBAR_RIGHT_STORAGE_KEY = 'echo:right-sidebar-collapsed';

	let chatSessions = $state<ChatSession[]>(
		importedChatSessions.map((session) => ({
			...session,
			pinned: session.pinned ?? false,
			folder: session.folder ?? 'General'
		}))
	);
	let folders = $state<string[]>(['General']);
	let activeChatId = $state(importedChatSessions[0]?.id ?? '');

	let recentChats = $derived(
		[...chatSessions]
			.sort((a, b) => Number(Boolean(b.pinned)) - Number(Boolean(a.pinned)))
			.map((session) => ({
				id: session.id,
				title: session.title,
				pinned: session.pinned,
				folder: session.folder ?? 'General'
			}))
	);
	let activeChat = $derived(chatSessions.find((session) => session.id === activeChatId));
	let messages = $derived(activeChat?.messages ?? []);

	let extractorPromise = $state<Promise<FeatureExtractor> | null>(null);

	async function getExtractor(): Promise<FeatureExtractor> {
		if (!extractorPromise) {
			extractorPromise = (async () => {
				const { pipeline } = await import('@xenova/transformers');
				return (await pipeline('feature-extraction', MODEL_NAME)) as FeatureExtractor;
			})();
		}
		return extractorPromise;
	}

	async function embed(text: string): Promise<number[]> {
		const extractor = await getExtractor();
		const output = await extractor(text, {
			pooling: 'mean',
			normalize: true
		});
		return Array.from(output.data);
	}

	function appendMessage(message: Omit<Message, 'id'>) {
		const withId: Message = {
			id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
			...message
		};

		chatSessions = chatSessions.map((session) =>
			session.id === activeChatId
				? { ...session, messages: [...session.messages, withId] }
				: session
		);
	}

	function refreshFoldersFromSessions() {
		const usedFolders = new Set(chatSessions.map((session) => session.folder ?? 'General'));
		folders = [
			'General',
			...folders.filter((folder) => folder !== 'General' && usedFolders.has(folder))
		];
	}

	async function sendMessage() {
		const text = input.trim();
		if (!text || loading) return;
		if (!activeChat) return;

		loading = true;
		input = '';
		appendMessage({ role: 'user', text });

		chatSessions = chatSessions.map((session) =>
			session.id === activeChatId && session.messages.length <= 1
				? { ...session, title: text.slice(0, 28) || 'New chat' }
				: session
		);

		try {
			const vector = await embed(text);
			const response = await fetch(`${BACKEND_URL}/chat`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ text, vector, model: selectedModel })
			});

			if (!response.ok) {
				throw new Error(`Request failed: ${response.status}`);
			}

			const payload: ChatApiResponse = await response.json();
			appendMessage({
				role: 'assistant',
				text: payload.answer,
				source: payload.source,
				showCard: true
			});
		} catch {
			appendMessage({
				role: 'assistant',
				text: 'Server unavailable. Start backend on port 8080 and try again.',
				source: 'CLOUD'
			});
		} finally {
			loading = false;
		}
	}

	function newChat() {
		const id = `chat-${Date.now()}`;
		chatSessions = [
			{ id, title: 'New chat', messages: [], folder: 'General', pinned: false },
			...chatSessions
		].slice(0, 20);
		activeChatId = id;
		if (isMobile) {
			leftSidebarCollapsed = true;
		}
		inputFocusTrigger += 1;
	}

	function selectChat(id: string) {
		activeChatId = id;
		if (isMobile) {
			leftSidebarCollapsed = true;
		}
		inputFocusTrigger += 1;
	}

	function renameChat(id: string) {
		const target = chatSessions.find((chat) => chat.id === id);
		if (!target) return;
		const nextTitle = window.prompt('Rename chat', target.title)?.trim();
		if (!nextTitle) return;
		chatSessions = chatSessions.map((chat) =>
			chat.id === id ? { ...chat, title: nextTitle } : chat
		);
	}

	function deleteChat(id: string) {
		const target = chatSessions.find((chat) => chat.id === id);
		if (!target) return;
		const confirmed = window.confirm(`Delete "${target.title}"?`);
		if (!confirmed) return;

		chatSessions = chatSessions.filter((chat) => chat.id !== id);
		if (activeChatId === id) {
			activeChatId = chatSessions[0]?.id ?? '';
		}
		refreshFoldersFromSessions();
	}

	function togglePinChat(id: string) {
		chatSessions = chatSessions.map((chat) =>
			chat.id === id ? { ...chat, pinned: !chat.pinned } : chat
		);
	}

	function moveChatToFolder(id: string) {
		const target = chatSessions.find((chat) => chat.id === id);
		if (!target) return;
		const suggested = target.folder ?? 'General';
		const nextFolder = window.prompt('Move to folder', suggested)?.trim();
		if (!nextFolder) return;

		chatSessions = chatSessions.map((chat) =>
			chat.id === id ? { ...chat, folder: nextFolder } : chat
		);
		if (!folders.includes(nextFolder)) {
			folders = [...folders, nextFolder];
		}
	}

	function createFolder() {
		const nextFolder = window.prompt('Folder name')?.trim();
		if (!nextFolder) return;
		if (folders.includes(nextFolder)) return;
		folders = [...folders, nextFolder];
	}

	function toggleTheme() {
		isDark = !isDark;
		document.documentElement.classList.toggle('dark', isDark);
	}

	function toggleLeftSidebar() {
		if (isMobile && leftSidebarCollapsed) {
			rightSidebarCollapsed = true;
		}
		leftSidebarCollapsed = !leftSidebarCollapsed;
	}

	function toggleRightSidebar() {
		if (isMobile && rightSidebarCollapsed) {
			leftSidebarCollapsed = true;
		}
		rightSidebarCollapsed = !rightSidebarCollapsed;
	}

	function closeLeftSidebar() {
		leftSidebarCollapsed = true;
	}

	function closeRightSidebar() {
		rightSidebarCollapsed = true;
	}

	function openChatSearch() {
		const query = window.prompt('Search chats by title')?.trim().toLowerCase();
		if (!query) return;
		const match = recentChats.find((chat) => chat.title.toLowerCase().includes(query));
		if (!match) {
			window.alert('No matching chat found.');
			return;
		}
		selectChat(match.id);
	}

	function handleShortcuts(event: KeyboardEvent) {
		const withCommand = event.ctrlKey || event.metaKey;
		if (!withCommand) return;

		const key = event.key.toLowerCase();
		if (key === 'b') {
			event.preventDefault();
			toggleLeftSidebar();
			return;
		}

		if (key === 'k') {
			event.preventDefault();
			openChatSearch();
		}
	}

	function handleViewportChange() {
		if (typeof window === 'undefined') return;
		isMobile = window.innerWidth < 1024;

		if (isMobile) {
			leftSidebarCollapsed = true;
			rightSidebarCollapsed = true;
			return;
		}

		const storedLeft = window.localStorage.getItem(SIDEBAR_LEFT_STORAGE_KEY);
		const storedRight = window.localStorage.getItem(SIDEBAR_RIGHT_STORAGE_KEY);

		if (storedLeft === null) {
			leftSidebarCollapsed = false;
		} else {
			leftSidebarCollapsed = storedLeft === '1';
		}
		if (storedRight === null) {
			rightSidebarCollapsed = false;
		} else {
			rightSidebarCollapsed = storedRight === '1';
		}
	}

	function startRightResize(event: MouseEvent) {
		if (isMobile || rightSidebarCollapsed) return;
		event.preventDefault();

		const startX = event.clientX;
		const startWidth = rightPaneWidth;

		const onMove = (moveEvent: MouseEvent) => {
			const delta = startX - moveEvent.clientX;
			rightPaneWidth = Math.min(520, Math.max(280, startWidth + delta));
		};

		const onUp = () => {
			window.removeEventListener('mousemove', onMove);
			window.removeEventListener('mouseup', onUp);
			rightResizeCleanup = null;
		};

		window.addEventListener('mousemove', onMove);
		window.addEventListener('mouseup', onUp);
		rightResizeCleanup = onUp;
	}

	function handleScrollToBottom() {
		chatCenterPaneRef?.scrollToBottom?.();
	}

	onMount(() => {
		handleViewportChange();
		window.addEventListener('resize', handleViewportChange);
		window.addEventListener('keydown', handleShortcuts);

		const derivedFolders = Array.from(
			new Set(chatSessions.map((session) => session.folder ?? 'General'))
		);
		folders = ['General', ...derivedFolders.filter((folder) => folder !== 'General')];

		return () => {
			window.removeEventListener('resize', handleViewportChange);
			window.removeEventListener('keydown', handleShortcuts);
			rightResizeCleanup?.();
		};
	});

	onDestroy(() => {
		rightResizeCleanup?.();
	});

	$effect(() => {
		document.documentElement.classList.toggle('dark', isDark);
		void getExtractor();
	});

	$effect(() => {
		if (typeof window === 'undefined') return;
		if (isMobile) return;
		window.localStorage.setItem(SIDEBAR_LEFT_STORAGE_KEY, leftSidebarCollapsed ? '1' : '0');
		window.localStorage.setItem(SIDEBAR_RIGHT_STORAGE_KEY, rightSidebarCollapsed ? '1' : '0');
	});
</script>

<div class="h-screen w-full bg-background text-foreground">
	<div class="flex h-full">
		{#if isMobile}
			{#if !leftSidebarCollapsed}
				<button
					type="button"
					onclick={closeLeftSidebar}
					class="fixed inset-0 z-30 bg-background/70"
					aria-label="Close chats drawer"
				></button>
			{/if}
			<div
				class={`fixed inset-y-0 left-0 z-40 h-full w-[260px] border-r border-border bg-card transition-transform duration-300 ease-in-out ${
					leftSidebarCollapsed ? '-translate-x-full' : 'translate-x-0'
				}`}
			>
				<LeftSidebar
					{recentChats}
					{folders}
					{activeChatId}
					{isDark}
					onNewChat={newChat}
					onSelectChat={selectChat}
					onToggleTheme={toggleTheme}
					onRenameChat={renameChat}
					onDeleteChat={deleteChat}
					onTogglePinChat={togglePinChat}
					onMoveChatToFolder={moveChatToFolder}
					onCreateFolder={createFolder}
				/>
			</div>
		{:else}
			<div
				class={`h-full overflow-hidden transition-all duration-300 ease-in-out ${
					leftSidebarCollapsed
						? 'w-0 -translate-x-full opacity-0'
						: 'w-[260px] translate-x-0 opacity-100'
				}`}
			>
				<LeftSidebar
					{recentChats}
					{folders}
					{activeChatId}
					{isDark}
					onNewChat={newChat}
					onSelectChat={selectChat}
					onToggleTheme={toggleTheme}
					onRenameChat={renameChat}
					onDeleteChat={deleteChat}
					onTogglePinChat={togglePinChat}
					onMoveChatToFolder={moveChatToFolder}
					onCreateFolder={createFolder}
				/>
			</div>
			{#if !leftSidebarCollapsed}
				<Separator orientation="vertical" />
			{/if}
		{/if}

		<div class="relative flex min-w-0 flex-1">
			<div class="pointer-events-none absolute top-4 left-4 z-30">
				<button
					type="button"
					onclick={toggleLeftSidebar}
					class="pointer-events-auto inline-flex h-9 w-9 items-center justify-center rounded-full border border-border bg-background text-sm"
					aria-label={leftSidebarCollapsed ? 'Show left sidebar' : 'Hide left sidebar'}
					title={leftSidebarCollapsed ? 'Show chats' : 'Hide chats'}
				>
					{#if leftSidebarCollapsed}
						<ChevronRight size={16} />
					{:else}
						<ChevronLeft size={16} />
					{/if}
				</button>
			</div>
			<div class="pointer-events-none absolute top-4 right-4 z-30">
				<button
					type="button"
					onclick={toggleRightSidebar}
					class="pointer-events-auto inline-flex h-9 w-9 items-center justify-center rounded-full border border-border bg-background text-sm"
					aria-label={rightSidebarCollapsed ? 'Show right sidebar' : 'Hide right sidebar'}
					title={rightSidebarCollapsed ? 'Show workspace' : 'Hide workspace'}
				>
					{#if rightSidebarCollapsed}
						<ChevronLeft size={16} />
					{:else}
						<ChevronRight size={16} />
					{/if}
				</button>
			</div>
			<ChatCenterPane bind:this={chatCenterPaneRef} {messages} {loading} />
			<FloatingInputBar
				bind:input
				bind:selectedModel
				focusTrigger={inputFocusTrigger}
				disabled={loading}
				onSubmit={sendMessage}
				onScrollToBottom={handleScrollToBottom}
			/>
		</div>

		{#if isMobile}
			{#if !rightSidebarCollapsed}
				<button
					type="button"
					onclick={closeRightSidebar}
					class="fixed inset-0 z-30 bg-background/70"
					aria-label="Close workspace drawer"
				></button>
			{/if}
			<div
				class={`fixed inset-y-0 right-0 z-40 h-full w-[85vw] max-w-[360px] border-l border-border bg-card transition-transform duration-300 ease-in-out ${
					rightSidebarCollapsed ? 'translate-x-full' : 'translate-x-0'
				}`}
			>
				<RightWorkspacePane />
			</div>
		{:else}
			{#if !rightSidebarCollapsed}
				<button
					type="button"
					class="w-1 cursor-col-resize bg-border/70 transition-colors hover:bg-border"
					onmousedown={startRightResize}
					aria-label="Resize right workspace pane"
				></button>
			{/if}
			<div
				class={`h-full shrink-0 overflow-hidden transition-all duration-300 ease-in-out ${
					rightSidebarCollapsed ? 'translate-x-full opacity-0' : 'translate-x-0 opacity-100'
				}`}
				style={`width:${rightSidebarCollapsed ? 0 : rightPaneWidth}px;`}
			>
				<RightWorkspacePane />
			</div>
		{/if}
	</div>
</div>
