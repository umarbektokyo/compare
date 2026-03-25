<script lang="ts">
	import { page } from '$app/stores';
	import { api, type Item, type Room } from '$lib/api';
	import { getAuth } from '$lib/auth.svelte';
	import { onMount } from 'svelte';

	const auth = getAuth();
	let roomId = $derived(Number($page.params.id));
	let room = $state<Room | null>(null);
	let isOwner = $derived(room !== null && auth.isOwner(room.owner_id));
	let items = $state<Item[]>([]);
	let showForm = $state(false);
	let newTitle = $state('');
	let newDesc = $state('');
	let newImage = $state('');
	let loading = $state(true);
	let tab = $state<'leaderboard' | 'history'>('leaderboard');
	let history = $state<any[]>([]);

	onMount(async () => {
		await loadData();
	});

	async function loadData() {
		loading = true;
		const [r, i] = await Promise.all([api.getRoom(roomId), api.getItems(roomId)]);
		room = r;
		items = i;
		loading = false;
	}

	async function addItem() {
		if (!newTitle.trim()) return;
		await api.addItem(roomId, newTitle.trim(), newDesc.trim(), newImage.trim());
		newTitle = '';
		newDesc = '';
		newImage = '';
		showForm = false;
		items = await api.getItems(roomId);
	}

	async function deleteItem(id: number) {
		if (!confirm('Delete this item?')) return;
		await api.deleteItem(id);
		items = await api.getItems(roomId);
	}

	async function loadHistory() {
		tab = 'history';
		history = await api.getHistory(roomId);
	}

	function getRankStyle(index: number) {
		if (index === 0) return `color: var(--gold)`;
		if (index === 1) return `color: var(--silver)`;
		if (index === 2) return `color: var(--bronze)`;
		return `color: var(--text-muted)`;
	}

	function getRankLabel(index: number) {
		if (index === 0) return '\u{1F947}';
		if (index === 1) return '\u{1F948}';
		if (index === 2) return '\u{1F949}';
		return `#${index + 1}`;
	}

	function winRate(item: Item) {
		if (item.matches === 0) return '-';
		return Math.round((item.wins / item.matches) * 100) + '%';
	}

	function confidence(item: Item) {
		// RD 350 = 0% confident (new), RD 50 = 100% confident (established)
		const rd = item.rd ?? 350;
		return Math.round(((350 - rd) / 300) * 100);
	}
</script>

{#if loading}
	<div class="container">
		<div class="skeleton-header"></div>
		<div class="skeleton-row"></div>
		<div class="skeleton-row"></div>
	</div>
{:else if room}
	<div class="container">
		<header class="room-header">
			<a href="/" class="back-link">
				<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M15 18l-6-6 6-6"/></svg>
			</a>
			<div class="room-title-area">
				<h1>{room.name}</h1>
				{#if room.description}
					<p class="desc">{room.description}</p>
				{/if}
			</div>
			{#if items.length >= 2}
				<a href="/room/{roomId}/play" class="play-btn">Play</a>
			{/if}
		</header>

		<div class="tabs">
			<button class="tab" class:active={tab === 'leaderboard'} onclick={() => tab = 'leaderboard'}>
				Rankings
			</button>
			<button class="tab" class:active={tab === 'history'} onclick={loadHistory}>
				Activity
			</button>
		</div>

		{#if tab === 'leaderboard'}
			<div class="section-bar">
				<span class="count">{items.length} item{items.length !== 1 ? 's' : ''}{room?.owner_name ? ` by ${room.owner_name}` : ''}</span>
				{#if isOwner}
					<button class="btn btn-ghost" onclick={() => showForm = !showForm}>
						{showForm ? 'Cancel' : '+ Add'}
					</button>
				{/if}
			</div>

			{#if showForm && isOwner}
				<form class="add-form" onsubmit={(e) => { e.preventDefault(); addItem(); }} style="animation: fadeIn 0.25s ease">
					<input class="input" placeholder="Title" bind:value={newTitle} autofocus />
					<input class="input" placeholder="Description (optional)" bind:value={newDesc} />
					<input class="input" placeholder="Image URL (optional)" bind:value={newImage} />
					{#if newImage}
						<img src={newImage} alt="preview" class="img-preview" />
					{/if}
					<button class="btn btn-primary" type="submit" style="align-self: flex-end; padding: 8px 24px;">Add Item</button>
				</form>
			{/if}

			{#if items.length === 0}
				<div class="empty-state">
					<p class="empty-title">Nothing here yet</p>
					<p class="empty-sub">Add at least 2 items to start comparing</p>
				</div>
			{:else}
				<div class="leaderboard">
					{#each items as item, i}
						<div class="lb-row" style="animation: fadeIn 0.2s ease {i * 0.03}s both">
							<div class="rank" class:rank-1={i === 0} class:rank-2={i === 1} class:rank-3={i === 2}>
								{#if i < 3}
									<span class="rank-medal">{getRankLabel(i)}</span>
								{:else}
									<span class="rank-num">{i + 1}</span>
								{/if}
							</div>
							<div class="lb-avatar">
								{#if item.image_url}
									<img src={item.image_url} alt={item.title} />
								{:else}
									<span>{item.title.charAt(0).toUpperCase()}</span>
								{/if}
							</div>
							<div class="lb-info">
								<span class="lb-title">{item.title}</span>
								{#if item.description}
									<span class="lb-desc">{item.description}</span>
								{:else}
									<span class="lb-desc">{item.wins}W {item.matches - item.wins}L &middot; {winRate(item)}</span>
								{/if}
							</div>
							<div class="lb-elo">
								<span class="elo-num">{Math.round(item.elo)}</span>
								<span class="elo-conf" title="Rating confidence: {confidence(item)}%">&plusmn;{Math.round(item.rd ?? 350)}</span>
							</div>
							{#if isOwner}
								<button class="del-btn" onclick={() => deleteItem(item.id)} title="Remove">
									<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
								</button>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		{:else}
			<div class="history">
				{#if history.length === 0}
					<div class="empty-state">
						<p class="empty-title">No activity yet</p>
						<p class="empty-sub">Play some matches to see history</p>
					</div>
				{:else}
					{#each history as match, i}
						<div class="history-row" style="animation: fadeIn 0.15s ease {i * 0.02}s both">
							<div class="h-main">
								<span class="h-winner">{match.winner}</span>
								<span class="h-vs">beat</span>
								<span class="h-loser">{match.item_a === match.winner ? match.item_b : match.item_a}</span>
							</div>
							<span class="h-elo">+{Math.round(match.elo_change * 10) / 10}</span>
						</div>
					{/each}
				{/if}
			</div>
		{/if}
	</div>
{/if}

<style>
	.room-header {
		display: flex;
		align-items: center;
		gap: 12px;
		margin-bottom: 24px;
		min-height: 44px;
	}
	.back-link {
		color: var(--text);
		display: flex;
		align-items: center;
		padding: 4px;
		flex-shrink: 0;
	}
	.room-title-area {
		flex: 1;
		min-width: 0;
	}
	h1 {
		font-size: 20px;
		font-weight: 700;
		letter-spacing: -0.3px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.desc {
		color: var(--text-tertiary);
		font-size: 13px;
		margin-top: 1px;
	}
	.play-btn {
		padding: 8px 20px;
		border-radius: 8px;
		background: var(--ig-gradient);
		font-weight: 600;
		font-size: 14px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		flex-shrink: 0;
		transition: transform 0.15s, opacity 0.15s;
	}
	.play-btn:hover {
		transform: scale(1.08);
	}
	.play-btn:active {
		transform: scale(0.95);
	}
	.tabs {
		display: flex;
		border-bottom: 1px solid var(--border);
		margin-bottom: 4px;
	}
	.tab {
		flex: 1;
		padding: 14px 0;
		color: var(--text-tertiary);
		font-weight: 600;
		font-size: 14px;
		text-align: center;
		border-bottom: 1px solid transparent;
		margin-bottom: -1px;
		transition: all 0.15s;
	}
	.tab:hover {
		color: var(--text-secondary);
	}
	.tab.active {
		color: var(--text);
		border-bottom-color: var(--text);
	}
	.section-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 0;
	}
	.count {
		color: var(--text-tertiary);
		font-size: 13px;
		font-weight: 500;
	}
	.add-form {
		background: var(--bg-card);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 14px;
		display: flex;
		flex-direction: column;
		gap: 8px;
		margin-bottom: 12px;
	}
	.img-preview {
		width: 56px;
		height: 56px;
		object-fit: cover;
		border-radius: var(--radius-sm);
	}
	.leaderboard {
		display: flex;
		flex-direction: column;
	}
	.lb-row {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 2px;
		border-bottom: 1px solid var(--border);
		transition: background 0.15s;
	}
	.lb-row:last-child {
		border-bottom: none;
	}
	.lb-row:hover {
		background: var(--bg-elevated);
	}
	.rank {
		width: 32px;
		text-align: center;
		flex-shrink: 0;
	}
	.rank-medal {
		font-size: 20px;
	}
	.rank-num {
		font-size: 14px;
		font-weight: 600;
		color: var(--text-tertiary);
	}
	.lb-avatar {
		width: 44px;
		height: 44px;
		border-radius: 50%;
		overflow: hidden;
		flex-shrink: 0;
		background: var(--bg-card);
		border: 1px solid var(--border);
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.lb-avatar img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.lb-avatar span {
		font-weight: 600;
		font-size: 16px;
		color: var(--text-secondary);
	}
	.lb-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 1px;
	}
	.lb-title {
		font-weight: 600;
		font-size: 14px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.lb-desc {
		color: var(--text-tertiary);
		font-size: 12px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.lb-elo {
		text-align: right;
		flex-shrink: 0;
	}
	.elo-num {
		font-weight: 700;
		font-size: 16px;
		display: block;
	}
	.elo-conf {
		font-size: 11px;
		color: var(--text-tertiary);
		font-weight: 500;
		cursor: help;
	}
	.del-btn {
		color: var(--text-tertiary);
		padding: 6px;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: all 0.15s;
		opacity: 0;
	}
	.lb-row:hover .del-btn {
		opacity: 1;
	}
	.del-btn:hover {
		color: var(--red);
		background: rgba(237, 73, 86, 0.1);
	}
	.empty-state {
		text-align: center;
		padding: 60px 20px;
	}
	.empty-title {
		font-size: 15px;
		font-weight: 600;
		margin-bottom: 4px;
	}
	.empty-sub {
		color: var(--text-tertiary);
		font-size: 13px;
	}
	.history {
		display: flex;
		flex-direction: column;
	}
	.history-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 4px;
		border-bottom: 1px solid var(--border);
		font-size: 14px;
	}
	.history-row:last-child {
		border-bottom: none;
	}
	.h-main {
		display: flex;
		align-items: center;
		gap: 6px;
		flex: 1;
		min-width: 0;
	}
	.h-winner {
		font-weight: 600;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.h-vs {
		color: var(--text-tertiary);
		font-size: 13px;
		flex-shrink: 0;
	}
	.h-loser {
		color: var(--text-secondary);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.h-elo {
		color: var(--green);
		font-weight: 600;
		font-size: 13px;
		flex-shrink: 0;
		margin-left: 12px;
	}
	.skeleton-header {
		height: 44px;
		background: linear-gradient(90deg, var(--bg-card) 25%, var(--bg-card-hover) 50%, var(--bg-card) 75%);
		background-size: 400px 100%;
		border-radius: var(--radius-sm);
		animation: shimmer 1.5s infinite;
		margin-bottom: 20px;
	}
	.skeleton-row {
		height: 60px;
		background: linear-gradient(90deg, var(--bg-card) 25%, var(--bg-card-hover) 50%, var(--bg-card) 75%);
		background-size: 400px 100%;
		border-radius: var(--radius-sm);
		animation: shimmer 1.5s infinite;
		margin-bottom: 8px;
	}
</style>