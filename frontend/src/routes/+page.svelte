<script lang="ts">
	import { api, type Room } from '$lib/api';
	import { getAuth } from '$lib/auth.svelte';
	import { onMount } from 'svelte';

	const auth = getAuth();
	let rooms = $state<Room[]>([]);
	let showForm = $state(false);
	let newName = $state('');
	let newDesc = $state('');
	let newImage = $state('');
	let loading = $state(true);
	let createError = $state('');

	onMount(async () => {
		await loadRooms();
	});

	async function loadRooms() {
		loading = true;
		rooms = await api.getRooms();
		loading = false;
	}

	async function createRoom() {
		if (!newName.trim()) return;
		createError = '';
		try {
			await api.createRoom(newName.trim(), newDesc.trim(), newImage.trim());
			newName = '';
			newDesc = '';
			newImage = '';
			showForm = false;
			await loadRooms();
		} catch (e: any) {
			createError = e.message;
		}
	}

	async function deleteRoom(id: number) {
		if (!confirm('Delete this room and all its items?')) return;
		try {
			await api.deleteRoom(id);
			await loadRooms();
		} catch {}
	}
</script>

<div class="container">
	{#if loading}
		<div class="loading">
			<div class="skeleton-card"></div>
			<div class="skeleton-card"></div>
		</div>
	{:else}
		<div class="page-header">
			<div>
				<h1>Rooms</h1>
				<p class="subtitle">{rooms.length} comparison room{rooms.length !== 1 ? 's' : ''}</p>
			</div>
			{#if auth.loggedIn}
				<button class="btn btn-primary" onclick={() => { showForm = !showForm; createError = ''; }}>
					{showForm ? 'Cancel' : 'Create'}
				</button>
			{/if}
		</div>

		{#if showForm}
			<form class="create-form" onsubmit={(e) => { e.preventDefault(); createRoom(); }} style="animation: fadeIn 0.25s ease">
				<input class="input" placeholder="Room name" bind:value={newName} autofocus />
				<input class="input" placeholder="Description (optional)" bind:value={newDesc} />
				<input class="input" placeholder="Image URL (optional)" bind:value={newImage} />
				{#if newImage}
					<img src={newImage} alt="preview" class="img-preview" />
				{/if}
				{#if createError}
					<p class="form-error">{createError}</p>
				{/if}
				<button class="btn btn-primary submit-btn" type="submit">Create Room</button>
			</form>
		{/if}

		{#if rooms.length === 0 && !showForm}
			<div class="empty-state">
				<div class="empty-ring">
					<span>VS</span>
				</div>
				<p class="empty-title">No rooms yet</p>
				<p class="empty-sub">Create a room to start comparing things</p>
			</div>
		{:else}
			<div class="rooms-list">
				{#each rooms as room, i}
					<a href="/room/{room.id}" class="room-item" style="animation: fadeIn 0.25s ease {i * 0.04}s both">
						<div class="room-avatar">
							{#if room.image_url}
								<img src={room.image_url} alt={room.name} />
							{:else}
								<span>{room.name.charAt(0).toUpperCase()}</span>
							{/if}
						</div>
						<div class="room-info">
							<span class="room-name">{room.name}</span>
							<span class="room-desc">
								{#if room.owner_name}by {room.owner_name} &middot; {/if}{room.item_count} item{room.item_count !== 1 ? 's' : ''}
							</span>
						</div>
						{#if auth.isOwner(room.owner_id)}
							<button class="delete-btn" onclick={(e) => { e.preventDefault(); deleteRoom(room.id); }} title="Delete">
								<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
							</button>
						{/if}
					</a>
				{/each}
			</div>
		{/if}
	{/if}
</div>

<style>
	.page-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 24px;
	}
	h1 {
		font-size: 24px;
		font-weight: 700;
		letter-spacing: -0.3px;
	}
	.subtitle {
		color: var(--text-tertiary);
		font-size: 14px;
		margin-top: 2px;
	}
	.create-form {
		background: var(--bg-card);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 16px;
		display: flex;
		flex-direction: column;
		gap: 10px;
		margin-bottom: 20px;
	}
	.submit-btn {
		align-self: flex-end;
		padding: 8px 24px;
	}
	.img-preview {
		width: 56px;
		height: 56px;
		object-fit: cover;
		border-radius: var(--radius-sm);
	}
	.form-error {
		color: var(--red);
		font-size: 13px;
	}
	.rooms-list {
		display: flex;
		flex-direction: column;
	}
	.room-item {
		display: flex;
		align-items: center;
		gap: 14px;
		padding: 14px 4px;
		border-bottom: 1px solid var(--border);
		transition: background 0.15s;
		border-radius: 0;
	}
	.room-item:last-child {
		border-bottom: none;
	}
	.room-item:hover {
		background: var(--bg-elevated);
	}
	.room-avatar {
		width: 52px;
		height: 52px;
		border-radius: 50%;
		background: var(--ig-gradient);
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		padding: 2px;
	}
	.room-avatar img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		border-radius: 50%;
	}
	.room-avatar span {
		width: 100%;
		height: 100%;
		background: var(--bg);
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-weight: 700;
		font-size: 18px;
		color: var(--text);
	}
	.room-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
	.room-name {
		font-weight: 600;
		font-size: 14px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.room-desc {
		color: var(--text-tertiary);
		font-size: 13px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.delete-btn {
		color: var(--text-tertiary);
		padding: 8px;
		border-radius: 50%;
		transition: all 0.15s;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.delete-btn:hover {
		color: var(--red);
		background: rgba(237, 73, 86, 0.1);
	}
	.loading {
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding-top: 20px;
	}
	.skeleton-card {
		height: 70px;
		background: linear-gradient(90deg, var(--bg-card) 25%, var(--bg-card-hover) 50%, var(--bg-card) 75%);
		background-size: 400px 100%;
		border-radius: var(--radius);
		animation: shimmer 1.5s infinite;
	}
	.empty-state {
		text-align: center;
		padding: 80px 20px 60px;
	}
	.empty-ring {
		width: 80px;
		height: 80px;
		border-radius: 50%;
		border: 2px solid var(--border);
		display: flex;
		align-items: center;
		justify-content: center;
		margin: 0 auto 16px;
		font-size: 22px;
		font-weight: 700;
		color: var(--text-tertiary);
	}
	.empty-title {
		font-size: 16px;
		font-weight: 600;
		margin-bottom: 4px;
	}
	.empty-sub {
		color: var(--text-tertiary);
		font-size: 14px;
	}
</style>
