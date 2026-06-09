export const apiModule = {
    async authorizedFetch(url, options = {}) {
        const headers = { ...(options.headers || {}) };
        const method = (options.method || 'GET').toUpperCase();

        if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(method)) {
            const csrfToken = this.getCookie('olivia_csrf');
            if (csrfToken) {
                headers['X-CSRF-Token'] = csrfToken;
            }
        }

        const res = await fetch(url, {
            ...options,
            headers,
            credentials: 'same-origin'
        });

        if (res.status === 401) {
            this.logout();
            throw new Error('Sessão expirada. Faça login novamente.');
        }

        return res;
    },

    async parseResponseSafely(res) {
        const raw = await res.text();
        if (!raw) return { data: null, raw: '' };

        try {
            return { data: JSON.parse(raw), raw };
        } catch {
            return { data: null, raw };
        }
    },

    getErrorMessage(payload, fallback = 'Erro desconhecido') {
        if (payload?.data && typeof payload.data === 'object') {
            if (payload.data.message) return payload.data.message;
            if (payload.data.error) return payload.data.error;
        }

        if (payload?.raw) {
            const plain = payload.raw.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim();
            if (plain) return plain.slice(0, 200);
        }

        return fallback;
    },
};
