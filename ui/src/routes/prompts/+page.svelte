<script>
	let prompts = $state([]);
	let models = $state([]);
	let error = $state('');
	let successMessage = $state('');

	let newPrompt = $state({ name: '', prompt_text: '', separator: '---', model: '' });
	let isEditModalOpen = $state(false);
	let editingPrompt = $state(null);

	$effect(() => {
		fetchPrompts();
		fetchModels();
	});

	function showSuccess(message) {
		successMessage = message;
		setTimeout(() => {
			successMessage = '';
		}, 3000); // Clear message after 3 seconds
	}

	async function fetchPrompts() {
		try {
			const res = await fetch('/api/prompts');
			if (!res.ok) throw new Error('Failed to fetch prompts');
			prompts = await res.json();
		} catch (e) {
			error = e.message;
		}
	}

	async function fetchModels() {
		try {
			const res = await fetch('/api/models');
			if (!res.ok) throw new Error('Failed to fetch models');
			models = await res.json();
			if (models.length > 0 && !newPrompt.model) {
				newPrompt.model = models[0];
			}
		} catch (e) {
			error = e.message;
		}
	}

	function handleClone(promptToClone) {
		newPrompt.name = `${promptToClone.name} (copy)`;
		newPrompt.prompt_text = promptToClone.prompt_text;
		newPrompt.separator = promptToClone.separator;
		newPrompt.model = promptToClone.model;
		document.getElementById('create-form-heading')?.scrollIntoView({ behavior: 'smooth' });
	}
	
	function openEditModal(promptToEdit) {
		editingPrompt = JSON.parse(JSON.stringify(promptToEdit));
		isEditModalOpen = true;
	}

	async function handleCreateSubmit(event) {
		event.preventDefault();
		error = '';
		try {
			const res = await fetch('/api/prompts', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(newPrompt)
			});
			if (!res.ok) {
				const errData = await res.json();
				throw new Error(errData.error || 'Failed to create prompt');
			}
			newPrompt = { name: '', prompt_text: '', separator: '---', model: models[0] };
			await fetchPrompts();
			showSuccess('Prompt created successfully!');
		} catch (e) {
			error = e.message;
		}
	}
	
	async function handleUpdateSubmit(event) {
		event.preventDefault();
		error = '';
		try {
			const res = await fetch('/api/prompts', {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(editingPrompt)
			});
			if (!res.ok) {
				const errData = await res.json();
				throw new Error(errData.error || 'Failed to update prompt');
			}
			isEditModalOpen = false;
			editingPrompt = null;
			await fetchPrompts();
			showSuccess('Prompt updated successfully!');
		} catch (e) {
			error = e.message;
		}
	}
</script>

<style>
	.prompt-list-item { display: grid; grid-template-columns: 1fr auto auto; gap: 1rem; align-items: center; margin-bottom: 1rem; }
	.prompt-info { overflow: hidden; }
	.prompt-info p { margin: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
	textarea { min-height: 25vh; font-family: monospace; }
	.message { padding: 1em; margin-bottom: 1em; border-radius: var(--border-radius); border: 1px solid; }
	.error { border-color: var(--pico-color-invalid); color: var(--pico-color-invalid); }
	.success { border-color: var(--pico-color-valid); color: var(--pico-color-valid); }
</style>

<svelte:head>
	<title>Manage Prompts</title>
</svelte:head>

<nav>
  <ul>
    <li><strong>Manage Prompts</strong></li>
  </ul>
  <ul>
    <li><a href="/" role="button" class="contrast">Back to Chat</a></li>
  </ul>
</nav>

<div class="grid">
	<section>
		<h2>Existing Prompts</h2>
		{#if successMessage}
			<div class="message success">{successMessage}</div>
		{/if}
		{#if prompts.length === 0}
			<p>No prompts created yet.</p>
		{/if}
		{#each prompts as p (p.id)}
			<article class="prompt-list-item">
				<div class="prompt-info">
					<strong>{p.name}</strong>
					<p><small>Model: {p.model.replace('models/', '')}</small></p>
				</div>
				<button class="secondary outline" onclick={() => openEditModal(p)}>Edit</button>
				<button class="contrast outline" onclick={() => handleClone(p)}>Clone</button>
			</article>
		{/each}
	</section>

	<section>
		<h2 id="create-form-heading">Create or Clone Prompt</h2>
		{#if error}<div class="message error">{error}</div>{/if}
		<form onsubmit={handleCreateSubmit}>
			<label for="new-name">Prompt Name</label>
			<input type="text" id="new-name" bind:value={newPrompt.name} required>
			<label for="new-text">Prompt Text</label>
			<textarea id="new-text" bind:value={newPrompt.prompt_text} required></textarea>
			<label for="new-separator">Separator</label>
			<input type="text" id="new-separator" bind:value={newPrompt.separator}>
			<label for="new-model">Model</label>
			<select id="new-model" bind:value={newPrompt.model} required>
				{#each models as model}
					<option value={model}>{model.replace('models/', '')}</option>
				{/each}
			</select>
			<button type="submit">Save New Prompt</button>
		</form>
	</section>
</div>

{#if isEditModalOpen}
<dialog open>
  <article>
    <header>
      <button aria-label="Close" class="close" onclick={() => isEditModalOpen = false}></button>
      <strong>Edit Prompt: {editingPrompt.name}</strong>
    </header>
	{#if error}<div class="message error" style="margin-bottom: 1em;">{error}</div>{/if}
    <form onsubmit={handleUpdateSubmit}>
		<label for="edit-name">Prompt Name</label>
		<input type="text" id="edit-name" bind:value={editingPrompt.name} required>
		<label for="edit-text">Prompt Text</label>
		<textarea id="edit-text" bind:value={editingPrompt.prompt_text} required></textarea>
		<label for="edit-separator">Separator</label>
		<input type="text" id="edit-separator" bind:value={editingPrompt.separator}>
		<label for="edit-model">Model</label>
		<select id="edit-model" bind:value={editingPrompt.model} required>
			{#each models as model}
				<option value={model}>{model.replace('models/', '')}</option>
			{/each}
		</select>
		<footer style="display: flex; justify-content: flex-end; gap: 1rem;">
			<button type="button" class="secondary" onclick={() => {isEditModalOpen = false; error = '';}}>Cancel</button>
			<button type="submit">Save Changes</button>
		</footer>
	</form>
  </article>
</dialog>
{/if}

