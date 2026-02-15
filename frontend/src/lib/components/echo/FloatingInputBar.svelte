<script lang="ts">
	import { tick } from 'svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Send } from '@lucide/svelte';

	export let input = '';
	export let disabled = false;
	export let selectedModel = 'gemini-2.5-flash-lite';
	export let onSubmit: () => void = () => {};

	const modelOptions = [
		{ value: 'gemini-2.5-flash-lite', label: 'Gemini 2.5 Flash Lite' },
		{ value: 'gemini-2.5-flash', label: 'Gemini 2.5 Flash' }
	];

	$: triggerContent =
		modelOptions.find((model) => model.value === selectedModel)?.label ?? 'Select model';

	let textareaEl: HTMLTextAreaElement;
	const maxInputHeight = 220;

	async function resizeInput() {
		await tick();
		if (!textareaEl) return;
		textareaEl.style.height = '0px';
		const nextHeight = Math.min(textareaEl.scrollHeight, maxInputHeight);
		textareaEl.style.height = `${nextHeight}px`;
		textareaEl.style.overflowY = textareaEl.scrollHeight > maxInputHeight ? 'auto' : 'hidden';
	}

	$: void resizeInput();

	function onEnter(event: KeyboardEvent) {
		if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			onSubmit();
		}
	}
</script>

<div class="pointer-events-none absolute inset-x-0 bottom-6 px-8">
	<div
		class="pointer-events-auto relative mx-auto flex w-full max-w-3xl flex-col items-stretch gap-2 rounded-2xl border border-border bg-background px-4 py-3 dark:bg-card"
	>
		<div
			class="pointer-events-none absolute right-4 bottom-full left-4 z-10 mb-[1px] h-12 bg-gradient-to-t from-background via-background/80 to-transparent"
		></div>

		<div class="pointer-events-auto relative z-20 flex w-full items-end gap-2 rounded-xl">
			<textarea
				bind:this={textareaEl}
				bind:value={input}
				on:keydown={onEnter}
				on:input={resizeInput}
				rows="1"
				placeholder="Ask Gemini..."
				class="min-h-9 flex-1 resize-none bg-transparent p-2 text-sm leading-6 outline-none"
			></textarea>
		</div>
		<div class="flex w-full items-center justify-end gap-2">
			<Select.Root type="single" name="chatModel" bind:value={selectedModel}>
				<Select.Trigger
					size="sm"
					class="h-8 w-[170px] border-none bg-card text-xs shadow-none dark:bg-card"
				>
					{triggerContent}
				</Select.Trigger>
				<Select.Content>
					<Select.Group>
						<Select.Label>Gemini Model</Select.Label>
						{#each modelOptions as model (model.value)}
							<Select.Item value={model.value} label={model.label}>
								{model.label}
							</Select.Item>
						{/each}
					</Select.Group>
				</Select.Content>
			</Select.Root>
			<Button
				on:click={onSubmit}
				disabled={disabled || !input.trim()}
				size="icon"
				className="rounded-full p-0"><Send size={16} /></Button
			>
		</div>
	</div>
</div>
