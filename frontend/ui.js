export const uiModule = {
    showNotification(message, type = 'success', duration = 4500) {
        const notification = document.getElementById('app-notification');
        if (!notification || !message) return;

        notification.classList.remove('hidden', 'success', 'error', 'visible');
        notification.textContent = message;
        const isError = type === 'error';
        notification.classList.add(isError ? 'error' : 'success');
        notification.setAttribute('aria-live', isError ? 'assertive' : 'polite');

        if (this.state.notificationTimeoutId) {
            clearTimeout(this.state.notificationTimeoutId);
        }
        if (this.state.notificationHideTimeoutId) {
            clearTimeout(this.state.notificationHideTimeoutId);
            this.state.notificationHideTimeoutId = null;
        }

        requestAnimationFrame(() => {
            notification.classList.add('visible');
        });

        this.state.notificationTimeoutId = setTimeout(() => {
            notification.classList.remove('visible');
            this.state.notificationHideTimeoutId = setTimeout(() => {
                notification.classList.add('hidden');
                this.state.notificationHideTimeoutId = null;
            }, 220);
            this.state.notificationTimeoutId = null;
        }, duration);
    },

    getCookie(name) {
        const escaped = name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        const match = document.cookie.match(new RegExp(`(?:^|; )${escaped}=([^;]*)`));
        return match ? decodeURIComponent(match[1]) : null;
    },

    escapeHtml(value) {
        return String(value ?? '')
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#039;');
    },

    syncAuthUI() {
        const isAuthenticatedView = this.state.authenticated && this.state.currentView !== 'login';
        const processBtn = document.getElementById('btn-process');
        const logoutBtn = document.getElementById('btn-logout');
        const indicator = document.getElementById('status-indicator');

        if (processBtn) processBtn.classList.toggle('hidden', !isAuthenticatedView);
        if (logoutBtn) logoutBtn.classList.toggle('hidden', !isAuthenticatedView);
        if (indicator) indicator.classList.toggle('hidden', !isAuthenticatedView || !this.state.statusPollingInterval);
    },
};
