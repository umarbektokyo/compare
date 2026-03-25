import { api } from './api';

let _user = $state<{ user_id: number; username: string } | null>(null);
let _loaded = $state(false);

export function getAuth() {
	return {
		get user() { return _user; },
		get loaded() { return _loaded; },
		get loggedIn() { return _user !== null; },

		async init() {
			const token = localStorage.getItem('token');
			if (!token) {
				_loaded = true;
				return;
			}
			try {
				_user = await api.me();
			} catch {
				localStorage.removeItem('token');
			}
			_loaded = true;
		},

		async login(username: string, password: string) {
			const res = await api.login(username, password);
			localStorage.setItem('token', res.token);
			_user = { user_id: res.user_id, username: res.username };
		},

		async register(username: string, password: string) {
			const res = await api.register(username, password);
			localStorage.setItem('token', res.token);
			_user = { user_id: res.user_id, username: res.username };
		},

		logout() {
			localStorage.removeItem('token');
			_user = null;
		},

		isOwner(ownerID: number) {
			return _user !== null && _user.user_id === ownerID;
		}
	};
}
