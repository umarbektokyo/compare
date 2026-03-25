<script lang="ts">
	import { page } from '$app/stores';
	import { api, type Item, type VoteResult } from '$lib/api';
	import { onMount } from 'svelte';

	let roomId = $derived(Number($page.params.id));
	let roomName = $state('');
	let itemA = $state<Item | null>(null);
	let itemB = $state<Item | null>(null);
	let phase = $state<'loading' | 'ready' | 'reveal' | 'result'>('loading');
	let winnerId = $state<number | null>(null);
	let result = $state<VoteResult | null>(null);
	let matchCount = $state(0);
	let error = $state('');

	onMount(async () => {
		try {
			const room = await api.getRoom(roomId);
			roomName = room.name;
			await loadPair();
		} catch (e: any) {
			error = e.message;
		}
	});

	async function loadPair() {
		phase = 'loading';
		winnerId = null;
		result = null;
		try {
			const pair = await api.getPair(roomId);
			itemA = pair.item_a;
			itemB = pair.item_b;
			phase = 'ready';
		} catch (e: any) {
			error = e.message;
		}
	}

	async function vote(winner: Item, loser: Item) {
		if (phase !== 'ready') return;
		winnerId = winner.id;
		phase = 'reveal';

		result = await api.vote(roomId, winner.id, loser.id);
		matchCount++;

		await new Promise(r => setTimeout(r, 600));
		phase = 'result';
	}

	async function next() {
		await loadPair();
	}
</script>

<div class="play-page">
	<div class="container">
		<header class="play-header">
			<a href="/room/{roomId}" class="back-link">
				<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M15 18l-6-6 6-6"/></svg>
			</a>
			<span class="header-title">{roomName}</span>
			<span class="match-count">{matchCount}</span>
		</header>

		{#if error}
			<div class="error-state">
				<p>{error}</p>
				<a href="/room/{roomId}" class="btn btn-primary" style="margin-top: 16px;">Go Back</a>
			</div>
		{:else if phase === 'loading'}
			<div class="loading-state">
				<div class="spinner"></div>
			</div>
		{:else if itemA && itemB}
			<p class="prompt">{phase === 'ready' ? 'Which do you prefer?' : phase === 'result' ? 'Result' : '...'}</p>

			<div class="arena">
				<button
					class="card-fighter"
					class:is-winner={phase !== 'ready' && winnerId === itemA.id}
					class:is-loser={phase !== 'ready' && winnerId === itemB.id}
					class:clickable={phase === 'ready'}
					onclick={() => vote(itemA!, itemB!)}
					disabled={phase !== 'ready'}
					style="animation: slideInLeft 0.4s cubic-bezier(0.16, 1, 0.3, 1)"
				>
					<div class="fighter-visual">
						{#if itemA.image_url}
							<img src={itemA.image_url} alt={itemA.title} />
						{:else}
							<div class="fighter-letter">{itemA.title.charAt(0).toUpperCase()}</div>
						{/if}
					</div>
					<div class="fighter-body">
						<h2>{itemA.title}</h2>
						{#if itemA.description}
							<p class="fighter-desc">{itemA.description}</p>
						{/if}
						<span class="fighter-elo">{Math.round(itemA.elo)} &plusmn;{Math.round(itemA.rd ?? 350)}</span>
					</div>
					{#if phase === 'result' && result}
						<div class="elo-badge" class:gain={winnerId === itemA.id} class:loss={winnerId !== itemA.id}>
							{winnerId === itemA.id ? '+' + result.winner_gain : '-' + result.loser_loss}
						</div>
					{/if}
				</button>

				<div class="vs-divider">
					<span class="vs-text" class:vs-active={phase === 'ready'}>VS</span>
				</div>

				<button
					class="card-fighter"
					class:is-winner={phase !== 'ready' && winnerId === itemB.id}
					class:is-loser={phase !== 'ready' && winnerId === itemA.id}
					class:clickable={phase === 'ready'}
					onclick={() => vote(itemB!, itemA!)}
					disabled={phase !== 'ready'}
					style="animation: slideInRight 0.4s cubic-bezier(0.16, 1, 0.3, 1)"
				>
					<div class="fighter-visual">
						{#if itemB.image_url}
							<img src={itemB.image_url} alt={itemB.title} />
						{:else}
							<div class="fighter-letter">{itemB.title.charAt(0).toUpperCase()}</div>
						{/if}
					</div>
					<div class="fighter-body">
						<h2>{itemB.title}</h2>
						{#if itemB.description}
							<p class="fighter-desc">{itemB.description}</p>
						{/if}
						<span class="fighter-elo">{Math.round(itemB.elo)} &plusmn;{Math.round(itemB.rd ?? 350)}</span>
					</div>
					{#if phase === 'result' && result}
						<div class="elo-badge" class:gain={winnerId === itemB.id} class:loss={winnerId !== itemB.id}>
							{winnerId === itemB.id ? '+' + result.winner_gain : '-' + result.loser_loss}
						</div>
					{/if}
				</button>
			</div>

			{#if phase === 'result' && result}
				{#if result.h2h_count > 1}
					<p class="h2h-note" style="animation: fadeIn 0.25s ease">
						Match #{result.h2h_count} between these two (diminishing impact)
					</p>
				{/if}
				<div class="next-area" style="animation: fadeIn 0.25s ease">
					<button class="next-btn" onclick={next}>
						Next
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M9 18l6-6-6-6"/></svg>
					</button>
				</div>
			{/if}
		{/if}
	</div>
</div>

<style>
	.play-page {
		min-height: calc(100vh - 60px);
		display: flex;
		align-items: flex-start;
	}
	.play-header {
		display: flex;
		align-items: center;
		margin-bottom: 32px;
	}
	.back-link {
		color: var(--text);
		display: flex;
		align-items: center;
		padding: 4px;
	}
	.header-title {
		flex: 1;
		text-align: center;
		font-weight: 600;
		font-size: 16px;
	}
	.match-count {
		background: var(--bg-card);
		border: 1px solid var(--border);
		color: var(--text-secondary);
		font-size: 12px;
		font-weight: 600;
		padding: 4px 10px;
		border-radius: 20px;
		min-width: 28px;
		text-align: center;
	}
	.prompt {
		text-align: center;
		font-size: 15px;
		font-weight: 600;
		color: var(--text-secondary);
		margin-bottom: 24px;
	}
	.arena {
		display: flex;
		align-items: stretch;
		gap: 12px;
	}
	.card-fighter {
		flex: 1;
		background: var(--bg-card);
		border: 1.5px solid var(--border);
		border-radius: var(--radius);
		overflow: hidden;
		display: flex;
		flex-direction: column;
		transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
		position: relative;
		color: var(--text);
		text-align: left;
	}
	.card-fighter.clickable {
		cursor: pointer;
	}
	.card-fighter.clickable:hover {
		border-color: var(--text-secondary);
		transform: translateY(-3px);
		box-shadow: var(--shadow-lg);
	}
	.card-fighter.clickable:active {
		transform: scale(0.98);
	}
	.card-fighter.is-winner {
		border-color: var(--green);
		animation: winPop 0.5s cubic-bezier(0.16, 1, 0.3, 1) forwards;
	}
	.card-fighter.is-loser {
		animation: loseFade 0.4s ease forwards;
	}
	.fighter-visual {
		width: 100%;
		aspect-ratio: 1;
		background: var(--bg-input);
		display: flex;
		align-items: center;
		justify-content: center;
		overflow: hidden;
	}
	.fighter-visual img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.fighter-letter {
		font-size: 56px;
		font-weight: 800;
		color: var(--text-tertiary);
		user-select: none;
	}
	.fighter-body {
		padding: 14px;
		display: flex;
		flex-direction: column;
		gap: 4px;
		flex: 1;
	}
	.fighter-body h2 {
		font-size: 16px;
		font-weight: 700;
		letter-spacing: -0.2px;
	}
	.fighter-desc {
		color: var(--text-tertiary);
		font-size: 13px;
		line-height: 1.4;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
	.fighter-elo {
		color: var(--text-tertiary);
		font-size: 12px;
		font-weight: 600;
		margin-top: auto;
		padding-top: 4px;
	}
	.elo-badge {
		position: absolute;
		top: 12px;
		right: 12px;
		font-size: 18px;
		font-weight: 800;
		animation: eloFloat 2s ease forwards;
		padding: 2px 8px;
		border-radius: 8px;
	}
	.elo-badge.gain {
		color: var(--green);
		background: rgba(46, 204, 113, 0.15);
	}
	.elo-badge.loss {
		color: var(--red);
		background: rgba(237, 73, 86, 0.15);
	}
	.vs-divider {
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		width: 40px;
	}
	.vs-text {
		font-size: 15px;
		font-weight: 800;
		color: var(--text-tertiary);
		letter-spacing: 1px;
		animation: vsAppear 0.4s ease 0.2s both;
	}
	.vs-text.vs-active {
		background: var(--ig-gradient);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		background-clip: text;
	}
	.h2h-note {
		text-align: center;
		color: var(--text-tertiary);
		font-size: 12px;
		margin-top: 16px;
	}
	.next-area {
		display: flex;
		justify-content: center;
		margin-top: 16px;
	}
	.next-btn {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		background: var(--link);
		color: white;
		font-weight: 600;
		font-size: 15px;
		padding: 12px 32px;
		border-radius: 10px;
		transition: opacity 0.15s, transform 0.1s;
	}
	.next-btn:hover {
		background: var(--link-hover);
	}
	.next-btn:active {
		transform: scale(0.96);
	}
	.loading-state {
		display: flex;
		justify-content: center;
		padding: 80px;
	}
	.spinner {
		width: 32px;
		height: 32px;
		border: 2.5px solid var(--border);
		border-top-color: var(--text-secondary);
		border-radius: 50%;
		animation: spin 0.7s linear infinite;
	}
	@keyframes spin {
		to { transform: rotate(360deg); }
	}
	.error-state {
		text-align: center;
		padding: 60px 20px;
		color: var(--text-secondary);
	}

	@media (max-width: 600px) {
		.arena {
			flex-direction: column;
			gap: 8px;
		}
		.vs-divider {
			width: auto;
			height: 32px;
		}
		.fighter-visual {
			aspect-ratio: 16/9;
		}
		.fighter-letter {
			font-size: 36px;
		}
	}
</style>