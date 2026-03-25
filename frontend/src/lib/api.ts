// In production the frontend is served by the Go server on the same origin,
// so we use a relative path. For local dev with separate vite server, set
// the VITE_API_URL env variable (e.g. http://localhost:8080/api).
const BASE = (typeof window !== 'undefined' && window.location.port === '8080')
	? '/api'
	: (import.meta.env.VITE_API_URL ?? '/api');

export interface Room {
	id: number;
	name: string;
	description: string;
	image_url: string;
	created_at: string;
	owner_id: number;
	owner_name: string;
	item_count: number;
}

export interface Item {
	id: number;
	title: string;
	description: string;
	image_url: string;
	elo: number;
	rd: number;
	matches: number;
	wins: number;
	created_at?: string;
}

export interface VoteResult {
	winner_new_elo: number;
	loser_new_elo: number;
	winner_gain: number;
	loser_loss: number;
	h2h_count: number;
	winner_rd: number;
	loser_rd: number;
}

export interface MatchHistory {
	id: number;
	item_a: string;
	item_b: string;
	winner: string;
	elo_change: number;
	created_at: string;
}

export interface AuthUser {
	token: string;
	user_id: number;
	username: string;
}

function getToken(): string | null {
	return localStorage.getItem('token');
}

async function fetchJSON<T>(url: string, opts?: RequestInit & { auth?: boolean }): Promise<T> {
	const headers: Record<string, string> = { 'Content-Type': 'application/json' };
	const token = getToken();
	if (token && opts?.auth !== false) {
		headers['Authorization'] = `Bearer ${token}`;
	}
	const res = await fetch(BASE + url, {
		...opts,
		headers: { ...headers, ...(opts?.headers as Record<string, string> || {}) }
	});
	if (!res.ok) {
		const err = await res.json().catch(() => ({ error: res.statusText }));
		throw new Error(err.error || res.statusText);
	}
	return res.json();
}

export const api = {
	// Auth
	register: (username: string, password: string) =>
		fetchJSON<AuthUser>('/auth/register', {
			method: 'POST',
			body: JSON.stringify({ username, password })
		}),
	login: (username: string, password: string) =>
		fetchJSON<AuthUser>('/auth/login', {
			method: 'POST',
			body: JSON.stringify({ username, password })
		}),
	me: () => fetchJSON<{ user_id: number; username: string }>('/auth/me'),

	// Rooms
	getRooms: () => fetchJSON<Room[]>('/rooms'),
	createRoom: (name: string, description: string, image_url: string) =>
		fetchJSON<{ id: number }>('/rooms', {
			method: 'POST',
			body: JSON.stringify({ name, description, image_url })
		}),
	getRoom: (id: number) => fetchJSON<Room>(`/rooms/${id}`),
	deleteRoom: (id: number) =>
		fetchJSON(`/rooms/${id}`, { method: 'DELETE' }),

	// Items
	getItems: (roomId: number) => fetchJSON<Item[]>(`/rooms/${roomId}/items`),
	addItem: (roomId: number, title: string, description: string, image_url: string) =>
		fetchJSON<{ id: number }>(`/rooms/${roomId}/items`, {
			method: 'POST',
			body: JSON.stringify({ title, description, image_url })
		}),
	deleteItem: (id: number) =>
		fetchJSON(`/items/${id}`, { method: 'DELETE' }),

	// Play
	getPair: (roomId: number) =>
		fetchJSON<{ item_a: Item; item_b: Item }>(`/rooms/${roomId}/pair`),
	vote: (roomId: number, winnerId: number, loserId: number) =>
		fetchJSON<VoteResult>('/vote', {
			method: 'POST',
			body: JSON.stringify({ room_id: roomId, winner_id: winnerId, loser_id: loserId })
		}),
	getHistory: (roomId: number) => fetchJSON<MatchHistory[]>(`/rooms/${roomId}/history`)
};
