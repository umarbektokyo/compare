<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { getAuth } from '$lib/auth.svelte';

	let { children } = $props();
	const auth = getAuth();

	let showAuth = $state(false);
	let authMode = $state<'login' | 'register'>('login');
	let username = $state('');
	let password = $state('');
	let authError = $state('');
	let authLoading = $state(false);
	let emailHint = $state('');

	onMount(() => {
		auth.init();
	});

	function handleUsernameInput() {
		if (username.includes('@')) {
			username = username.split('@')[0];
			emailHint = 'Just a username is needed — email removed for privacy';
		} else {
			emailHint = '';
		}
	}

	async function handleAuth(e: Event) {
		e.preventDefault();
		authError = '';
		authLoading = true;
		try {
			if (authMode === 'login') {
				await auth.login(username, password);
			} else {
				await auth.register(username, password);
			}
			showAuth = false;
			username = '';
			password = '';
		} catch (err: any) {
			authError = err.message;
		}
		authLoading = false;
	}
</script>

<svelte:head>
	<link rel="icon" href="/favicon.svg" type="image/svg+xml" />
</svelte:head>

<div class="app">
	<nav class="navbar">
		<div class="container nav-inner">
			<a href="/" class="logo">Compare</a>
			<div class="nav-right">
				{#if auth.loggedIn}
					<span class="nav-user">{auth.user?.username}</span>
					<button class="nav-btn" onclick={() => auth.logout()}>Log out</button>
				{:else}
					<button class="nav-btn nav-btn-primary" onclick={() => { showAuth = true; authMode = 'login'; authError = ''; }}>Log in</button>
				{/if}
			</div>
		</div>
	</nav>

	{#if showAuth}
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="overlay" onclick={() => showAuth = false} onkeydown={() => {}}>
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div class="auth-modal" onclick={(e) => e.stopPropagation()} onkeydown={() => {}}>
				<div class="auth-logo">Compare</div>

				<div class="auth-tabs">
					<button class="auth-tab" class:active={authMode === 'login'} onclick={() => { authMode = 'login'; authError = ''; }}>Log in</button>
					<button class="auth-tab" class:active={authMode === 'register'} onclick={() => { authMode = 'register'; authError = ''; }}>Sign up</button>
				</div>

				<form class="auth-form" onsubmit={handleAuth}>
					<input class="input" type="text" placeholder="Username" bind:value={username} oninput={handleUsernameInput} autocomplete="username" />
					{#if emailHint}
						<p class="email-hint">{emailHint}</p>
					{/if}
					<input class="input" type="password" placeholder="Password" bind:value={password} autocomplete={authMode === 'register' ? 'new-password' : 'current-password'} />
					{#if authError}
						<p class="auth-error">{authError}</p>
					{/if}
					<button class="btn btn-primary auth-submit" type="submit" disabled={authLoading}>
						{authLoading ? '...' : authMode === 'login' ? 'Log in' : 'Sign up'}
					</button>
				</form>

				<p class="auth-switch">
					{#if authMode === 'login'}
						Don't have an account? <button class="link-btn" onclick={() => { authMode = 'register'; authError = ''; }}>Sign up</button>
					{:else}
						Have an account? <button class="link-btn" onclick={() => { authMode = 'login'; authError = ''; }}>Log in</button>
					{/if}
				</p>
			</div>
		</div>
	{/if}

	<main>
		{@render children()}
	</main>
</div>

<style>
	.app {
		min-height: 100vh;
	}
	.navbar {
		border-bottom: 1px solid var(--border);
		height: 60px;
		display: flex;
		align-items: center;
		background: var(--bg);
		position: sticky;
		top: 0;
		z-index: 100;
	}
	.nav-inner {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
	}
	.logo {
		font-size: 22px;
		font-weight: 700;
		letter-spacing: -0.5px;
		background: var(--ig-gradient);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		background-clip: text;
	}
	.nav-right {
		display: flex;
		align-items: center;
		gap: 12px;
	}
	.nav-user {
		font-size: 14px;
		font-weight: 600;
	}
	.nav-btn {
		font-size: 13px;
		font-weight: 600;
		color: var(--text-secondary);
		padding: 6px 12px;
		border-radius: 8px;
		transition: all 0.15s;
	}
	.nav-btn:hover {
		color: var(--text);
	}
	.nav-btn-primary {
		background: var(--link);
		color: white;
	}
	.nav-btn-primary:hover {
		background: var(--link-hover);
		color: white;
	}
	main {
		padding: 24px 0 60px;
	}

	/* Auth modal */
	.overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.65);
		z-index: 200;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 20px;
	}
	.auth-modal {
		background: var(--bg-card);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 32px 28px;
		width: 100%;
		max-width: 360px;
		animation: fadeIn 0.2s ease;
	}
	.auth-logo {
		font-size: 28px;
		font-weight: 700;
		text-align: center;
		margin-bottom: 24px;
		background: var(--ig-gradient);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		background-clip: text;
	}
	.auth-tabs {
		display: flex;
		border-bottom: 1px solid var(--border);
		margin-bottom: 20px;
	}
	.auth-tab {
		flex: 1;
		padding: 10px 0;
		text-align: center;
		font-weight: 600;
		font-size: 14px;
		color: var(--text-tertiary);
		border-bottom: 1px solid transparent;
		margin-bottom: -1px;
		transition: all 0.15s;
	}
	.auth-tab.active {
		color: var(--text);
		border-bottom-color: var(--text);
	}
	.auth-form {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.email-hint {
		color: var(--link);
		font-size: 12px;
		margin-top: -4px;
	}
	.auth-error {
		color: var(--red);
		font-size: 13px;
		text-align: center;
	}
	.auth-submit {
		width: 100%;
		padding: 10px;
		margin-top: 4px;
		font-size: 14px;
	}
	.auth-submit:disabled {
		opacity: 0.6;
	}
	.auth-switch {
		text-align: center;
		font-size: 13px;
		color: var(--text-tertiary);
		margin-top: 20px;
	}
	.link-btn {
		color: var(--link);
		font-weight: 600;
		font-size: 13px;
	}
</style>
