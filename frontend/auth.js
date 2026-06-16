import { API_URL } from './constants.js';

export const authModule = {
    async checkSession() {
        try {
            const res = await fetch(`${API_URL}/auth/verify`, {
                method: 'GET',
                credentials: 'same-origin'
            });

            this.state.authenticated = res.ok;
        } catch (err) {
            console.error('Session check failed:', err);
            this.state.authenticated = false;
        }

        if (this.state.authenticated) {
            this.navigate('queue');
            return;
        }

        this.navigate('login');
    },

    async login() {
        const user = document.getElementById('login-user').value;
        const pass = document.getElementById('login-pass').value;

        try {
            const res = await fetch(`${API_URL}/login`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username: user, password: pass }),
                credentials: 'same-origin'
            });

            if (res.ok) {
                await res.json();
                this.state.authenticated = true;
                this.navigate('queue');
            } else {
                this.showNotification('Credenciais inválidas', 'error');
            }
        } catch (err) {
            console.error(err);
            this.showNotification('Erro ao realizar login', 'error');
        }
    },

    async logout() {
        this.state.authenticated = false;
        if (this.state.statusPollingInterval) {
            clearInterval(this.state.statusPollingInterval);
            this.state.statusPollingInterval = null;
        }
        this.state.currentExecution = null;
        this.closeStatusModal();

        try {
            await fetch(`${API_URL}/logout`, {
                method: 'POST',
                credentials: 'same-origin'
            });
        } catch (err) {
            console.error(err);
        }
        this.navigate('login');
    },
};
