<script lang="ts">
	import LeftSidebar from '$lib/components/echo/LeftSidebar.svelte';
	import ChatCenterPane from '$lib/components/echo/ChatCenterPane.svelte';
	import RightWorkspacePane from '$lib/components/echo/RightWorkspacePane.svelte';
	import FloatingInputBar from '$lib/components/echo/FloatingInputBar.svelte';

	// ui
	import Separator from '$lib/components/ui/Separator.svelte';

	// lib
	import type { Message, ChatApiResponse, FeatureExtractor } from '$lib/lib/chat';
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
	let chatCenterPaneRef = $state<{
		scrollToBottom?: () => void;
	} | null>(null);

	let chatSessions = $state(importedChatSessions);
	let activeChatId = $state(importedChatSessions[0]?.id ?? '');

	let recentChats = $derived(
		chatSessions.map((session) => ({ id: session.id, title: session.title }))
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
		chatSessions = [{ id, title: 'New chat', messages: [] }, ...chatSessions].slice(0, 8);
		activeChatId = id;
	}

	function selectChat(id: string) {
		activeChatId = id;
	}

	function toggleTheme() {
		isDark = !isDark;
		document.documentElement.classList.toggle('dark', isDark);
	}

	function toggleLeftSidebar() {
		leftSidebarCollapsed = !leftSidebarCollapsed;
	}

	function toggleRightSidebar() {
		rightSidebarCollapsed = !rightSidebarCollapsed;
	}

	function handleScrollToBottom() {
		chatCenterPaneRef?.scrollToBottom?.();
	}

	$effect(() => {
		document.documentElement.classList.toggle('dark', isDark);
		void getExtractor();
	});
</script>

<div class="h-screen w-full bg-background text-foreground">
	<div class="flex h-full">
		<div
			class={`h-full overflow-hidden transition-all duration-300 ease-in-out ${
				leftSidebarCollapsed
					? 'w-0 -translate-x-full opacity-0'
					: 'w-[260px] translate-x-0 opacity-100'
			}`}
		>
			<LeftSidebar
				{recentChats}
				{activeChatId}
				{isDark}
				onNewChat={newChat}
				onSelectChat={selectChat}
				onToggleTheme={toggleTheme}
			/>
		</div>
		{#if !leftSidebarCollapsed}
			<Separator orientation="vertical" />
		{/if}

		<div class="relative flex min-w-0 flex-1">
			<div class="pointer-events-none absolute top-4 left-4 z-30">
				<button
					type="button"
					onclick={toggleLeftSidebar}
					class="pointer-events-auto inline-flex h-9 w-9 items-center justify-center rounded-full border border-border bg-background text-sm"
					aria-label={leftSidebarCollapsed ? 'Show left sidebar' : 'Hide left sidebar'}
				>
					☰
				</button>
			</div>
			<div class="pointer-events-none absolute top-4 right-4 z-30">
				<button
					type="button"
					onclick={toggleRightSidebar}
					class="pointer-events-auto inline-flex h-9 w-9 items-center justify-center rounded-full border border-border bg-background text-sm"
					aria-label={rightSidebarCollapsed ? 'Show right sidebar' : 'Hide right sidebar'}
				>
					☰
				</button>
			</div>
			<ChatCenterPane bind:this={chatCenterPaneRef} {messages} {loading} />
			<FloatingInputBar
				bind:input
				bind:selectedModel
				disabled={loading}
				onSubmit={sendMessage}
				onScrollToBottom={handleScrollToBottom}
			/>
		</div>

		{#if !rightSidebarCollapsed}
			<Separator orientation="vertical" />
		{/if}
		<div
			class={`h-full overflow-hidden transition-all duration-300 ease-in-out ${
				rightSidebarCollapsed
					? 'w-0 translate-x-full opacity-0'
					: 'w-[360px] translate-x-0 opacity-100'
			}`}
		>
			<RightWorkspacePane />
		</div>
	</div>
</div>
