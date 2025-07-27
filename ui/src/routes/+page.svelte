<script>
	import { marked } from 'marked';

	let prompts = $state([]);
	let models = $state([]);
	let chats = $state([]);
	let selectedPromptId = $state('');
	let userInput = $state('');
	let currentOutput = $state('');
	let currentMetadata = $state(null);
	let isLoading = $state(false);
	let error = $state('');
	let isRerunModalOpen = $state(false);
	let rerunData = $state(null);

	$effect(() => {
		fetchPrompts();
		fetchModels();
		fetchChats();
	});

	// --- THE FIX IS HERE: Part 1 ---
	// This $effect hook correctly manipulates the global document body
	// to add/remove the 'loading' class for the cursor.
	// It runs only when the `isLoading` state changes.
	$effect(() => {
		if (isLoading) {
			document.body.classList.add('loading');
		} else {
			document.body.classList.remove('loading');
		}
	});

	async function fetchPrompts() { try { const res = await fetch('/api/prompts'); if (!res.ok) throw new Error('Failed to fetch prompts'); prompts = await res.json(); if (prompts.length > 0 && !selectedPromptId) { selectedPromptId = prompts[0].id; } } catch (e) { error = e.message; } }
	async function fetchModels() { try { const res = await fetch('/api/models'); if (!res.ok) throw new Error('Failed to fetch models'); models = await res.json(); } catch (e) { error = e.message; } }
	async function fetchChats() { try { const res = await fetch('/api/chats'); if (!res.ok) throw new Error('Failed to fetch chats'); chats = await res.json(); } catch (e) { error = e.message; } }

	async function handleSubmit(event, overrideModel = '') {
		event.preventDefault();
		if (!selectedPromptId || !userInput.trim()) {
			error = 'Please select a prompt and provide input.';
			return;
		}
		isLoading = true;
		error = '';
		currentOutput = '';
		currentMetadata = null;
		const body = { prompt_id: parseInt(selectedPromptId, 10), user_input: userInput, ...(overrideModel && { model: overrideModel }) };
		try {
			const res = await fetch('/api/generate', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
			if (!res.ok) {
				const errorText = await res.text();
				try { error = JSON.parse(errorText).error || errorText; } catch { error = errorText || `Request failed with status ${res.status}`; }
				return;
			}
			const data = await res.json();
			currentOutput = data.model_output.String;
			currentMetadata = data;
			await fetchChats();
		} catch (e) {
			error = e.message;
		} finally {
			isLoading = false;
			isRerunModalOpen = false;
		}
	}
	
	function openRerunModal(chat) {
		rerunData = { userInput: chat.user_input, promptId: chat.prompt_id.Int64, selectedModel: chat.model_used.String || models[0] };
		isRerunModalOpen = true;
	}

	function handleRerunSubmit(event) {
		selectedPromptId = rerunData.promptId;
		userInput = rerunData.userInput;
		handleSubmit(event, rerunData.selectedModel);
	}

	function loadChat(chat) {
		selectedPromptId = chat.prompt_id.Int64;
		userInput = chat.user_input;
		currentOutput = chat.model_output.String;
		currentMetadata = chat;
		error = chat.error_message.String;
	}
	
	const renderedOutput = $derived(marked(currentOutput || ''));
</script>

<style>
	:global(body.loading) { cursor: wait; }
	.grid { display: grid; grid-template-columns: 300px 1fr; gap: 2rem; align-items: start; }
	aside { height: 90vh; overflow-y: auto; position: sticky; top: 1rem; }
	aside ul { list-style: none; padding: 0; margin: 0; }
	aside li { padding: 0.5rem; margin-bottom: 0.5rem; border: 1px solid var(--muted-border-color); border-radius: var(--border-radius); }
	.history-prompt-info { font-size: 0.8em; color: var(--muted-color); word-break: break-all; margin-top: 0.5rem; line-height: 1.4; }
	.history-item-actions { font-size: 0.8em; text-align: right; margin-top: 0.5rem; }
	#user-input { min-height: 40vh; font-family: monospace; }
	.output-box { min-height: 200px; border: 1px solid var(--form-element-border-color); padding: 1rem; border-radius: var(--border-radius); background: var(--card-background-color); overflow-wrap: break-word; }
	.metadata { font-size: 0.8em; color: var(--muted-color); margin-top: 1rem; }
	.error { padding: 1em; margin-bottom: 1em; border-radius: var(--border-radius); border: 1px solid var(--pico-color-invalid); color: var(--pico-color-invalid); }
	button.link-style { background: none; border: none; padding: 0; margin: 0; color: var(--pico-primary); text-decoration: underline; cursor: pointer; text-align: left; font: inherit; width: 100%;}
</style>

<!-- --- THE FIX IS HERE: Part 2 --- -->
<!-- The invalid <svelte:head> block has been completely removed. -->
<svelte:head>
	<title>AI Tool</title>
</svelte:head>

<div class="grid">
	<aside>
		<a href="/prompts" role="button" class="contrast" style="width: 100%; margin-bottom: 1rem;">Manage Prompts</a>
		<h4>Chat History</h4>
		<ul>
			{#each chats as chat (chat.id)}
				<li>
					<button class="link-style" onclick={() => loadChat(chat)}>
						{chat.user_input.substring(0, 70)}{#if chat.user_input.length > 70}...{/if}
					</button>
					<div class="history-prompt-info">
						<strong>{new Date(chat.request_timestamp).toLocaleString()}</strong><br/>
						Using: "{chat.prompt_name.String || 'Unknown Prompt'}"<br/>
						Model: {(chat.model_used.String || 'Unknown Model').replace('models/', '')}<br/>
						{#if chat.input_token_count.Valid}
							Tokens: {chat.input_token_count.Int64} in, {chat.output_token_count.Int64} out
						{/if}
					</div>
					<div class="history-item-actions">
						<button class="link-style" onclick={() => openRerunModal(chat)} title="Re-run with a different model">Re-run...</button>
					</div>
				</li>
			{/each}
		</ul>
	</aside>
	
	<section>
		<h1>AI Tool</h1>
		<form onsubmit={handleSubmit}>
			<label for="prompt-select">Select a Saved Prompt</label>
			<select id="prompt-select" bind:value={selectedPromptId} required>
				{#if prompts.length === 0}
				<option disabled>Please create a prompt first</option>
				{/if}
				{#each prompts as p (p.id)}
					<option value={p.id}>{p.name}</option>
				{/each}
			</select>
			<label for="user-input">Your Input</label>
			<textarea id="user-input" bind:value={userInput} placeholder="Enter your text here..."></textarea>
			<button type="submit" aria-busy={isLoading} disabled={prompts.length === 0}>
				{#if isLoading}Submitting...{:else}Submit{/if}
			</button>
		</form>

		{#if error}<div class="error">{error}</div>{/if}

		<h4>Output</h4>
		<div class="output-box">{#if isLoading}<progress></progress>{:else if currentOutput}{@html renderedOutput}{:else}<p>Model output will appear here.</p>{/if}</div>
		
		{#if currentMetadata && !isLoading}
			<div class="metadata">
				<p><strong>Time Taken:</strong> {currentMetadata.time_taken_ms.Int64}ms | <strong>Input Tokens:</strong> {currentMetadata.input_token_count.Int64} | <strong>Output Tokens:</strong> {currentMetadata.output_token_count.Int64}</p>
			</div>
		{/if}
	</section>
</div>

{#if isRerunModalOpen}
<dialog open>
  <article>
    <header>
      <button aria-label="Close" class="close" onclick={() => isRerunModalOpen = false}></button>
      <strong>Re-run with a different model</strong>
    </header>
    <p style="word-break: break-all;"><strong>Input:</strong> {rerunData.userInput.substring(0, 200)}...</p>
	<form onsubmit={handleRerunSubmit}>
		<label for="rerun-model">Model</label>
		<select id="rerun-model" bind:value={rerunData.selectedModel} required>
			{#each models as model}
				<option value={model}>{model.replace('models/', '')}</option>
			{/each}
		</select>
		<footer style="display: flex; justify-content: flex-end; gap: 1rem;">
			<button type="button" class="secondary" onclick={() => isRerunModalOpen = false}>Cancel</button>
			<button type="submit" aria-busy={isLoading}>Re-run with selected model</button>
		</footer>
	</form>
  </article>
</dialog>
{/if}
