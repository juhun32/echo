<script lang="ts">
	import { onMount } from 'svelte';
	import LeftSidebar from '$lib/components/echo/LeftSidebar.svelte';
	import ChatCenterPane from '$lib/components/echo/ChatCenterPane.svelte';
	import RightWorkspacePane from '$lib/components/echo/RightWorkspacePane.svelte';
	import FloatingInputBar from '$lib/components/echo/FloatingInputBar.svelte';
	import Separator from '$lib/components/ui/Separator.svelte';

	type RecentChat = {
		id: string;
		title: string;
	};

	type Message = {
		id: string;
		role: 'user' | 'assistant';
		text: string;
		source?: 'CACHE' | 'CLOUD';
		showCard?: boolean;
	};

	type ChatSession = {
		id: string;
		title: string;
		messages: Message[];
	};

	type ChatApiResponse = {
		answer: string;
		source: 'CACHE' | 'CLOUD';
	};

	type ExtractorOutput = {
		data: Float32Array;
	};

	type FeatureExtractor = (
		text: string,
		options: { pooling: 'mean'; normalize: boolean }
	) => Promise<ExtractorOutput>;

	const API_URL = 'http://localhost:8080/chat';
	const MODEL_NAME = 'Xenova/all-MiniLM-L6-v2';

	let input = '';
	let loading = false;
	let isDark = true;
	let selectedModel = 'gemini-2.5-flash-lite';
	let leftSidebarCollapsed = false;
	let rightSidebarCollapsed = false;
	let chatSessions: ChatSession[] = [
		{ id: 'chat-1', title: 'How to center', messages: [] },
		{ id: 'chat-2', title: 'Pricing copy ideas', messages: [] },
		{ id: 'chat-3', title: 'Console debug output', messages: [] }
	];
	let activeChatId = chatSessions[0].id;

	$: recentChats = chatSessions.map((session) => ({ id: session.id, title: session.title }));
	$: activeChat = chatSessions.find((session) => session.id === activeChatId);
	$: messages = activeChat?.messages ?? [];

	let extractorPromise: Promise<FeatureExtractor> | null = null;

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
			const response = await fetch(API_URL, {
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

	onMount(() => {
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
					on:click={toggleLeftSidebar}
					class="pointer-events-auto inline-flex h-9 w-9 items-center justify-center rounded-full border border-border bg-background text-sm"
					aria-label={leftSidebarCollapsed ? 'Show left sidebar' : 'Hide left sidebar'}
				>
					☰
				</button>
			</div>
			<div class="pointer-events-none absolute top-4 right-4 z-30">
				<button
					type="button"
					on:click={toggleRightSidebar}
					class="pointer-events-auto inline-flex h-9 w-9 items-center justify-center rounded-full border border-border bg-background text-sm"
					aria-label={rightSidebarCollapsed ? 'Show right sidebar' : 'Hide right sidebar'}
				>
					☰
				</button>
			</div>
			<ChatCenterPane {messages} {loading} />
			<FloatingInputBar bind:input bind:selectedModel disabled={loading} onSubmit={sendMessage} />
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
