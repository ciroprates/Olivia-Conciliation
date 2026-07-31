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

    // ── Helpers de exibição do layout de abas (issue #6) ──

    formatCurrency(value) {
        return 'R$ ' + Number(value || 0).toFixed(2).replace('.', ',');
    },

    // txId é gerado no cliente a partir do rowIndex (a API não retorna id) —
    // ver convenção de `txId` na issue #6. Ex.: DIF-5, ES-34, REC-12, NRC-8.
    formatTxId(prefix, rowIndex) {
        return `${prefix}-${rowIndex}`;
    },

    // idParcela tem o formato `<hash>/<num>`; exibimos só `/<num>` no badge.
    parcelaSuffix(idParcela) {
        if (!idParcela) return '';
        const slash = String(idParcela).lastIndexOf('/');
        return slash >= 0 ? String(idParcela).slice(slash) : '';
    },

    // Δ entre o valor da DIF e o da candidata ES. A API não retorna score de
    // match; a proximidade de valor é o sinal visível (regra de matching: < 5,00).
    deltaHtml(difValue, esValue) {
        const d = Number(difValue || 0) - Number(esValue || 0);
        if (Math.abs(d) < 0.01) return '<span class="delta-zero">= exato</span>';
        const sign = d > 0 ? '+' : '−';
        const cls = d > 0 ? 'delta-pos' : 'delta-neg';
        return `<span class="${cls}">${sign}${this.formatCurrency(Math.abs(d))}</span>`;
    },

    categoryColor(cat) {
        const COLORS = {
            'Alimentação': '#fb923c', 'Lazer': '#a78bfa', 'Moradia': '#60a5fa',
            'Saúde': '#34d399', 'Transporte': '#f472b6', 'Outros': '#94a3b8'
        };
        return COLORS[cat] || '#94a3b8';
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
