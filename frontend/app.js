const API_URL = '/api';
const EXECUTION_API_URL = '/executions';
const EXECUTION_OPTIONS_STORAGE_KEY = 'olivia_execution_options_v1';
const DEFAULT_EXECUTION_OPTIONS = {
    excludeCategories: ['Same person transfer', 'Credit card payment'],
    startDate: '2026-01-15'
};
const DEFAULT_EXECUTION_PAYLOAD = {
    sheet: {
        enabled: true,
        tabName: 'Homologacao'
    },
    artifacts: {
        csvEnabled: false
    }
};

const app = {
    state: {
        authenticated: false,
        currentView: 'queue',
        conciliations: [],
        details: null,
        selectedCandidates: new Set(),
        currentExecution: null,
        executionHistory: [],
        statusPollingInterval: null,
        executionOptions: { ...DEFAULT_EXECUTION_OPTIONS }
    },

    async init() {
        await this.checkSession();
    },

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
            // Avoid showing a full HTML page in alerts.
            const plain = payload.raw.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim();
            if (plain) return plain.slice(0, 200);
        }

        return fallback;
    },

    getCookie(name) {
        const escaped = name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        const match = document.cookie.match(new RegExp(`(?:^|; )${escaped}=([^;]*)`));
        return match ? decodeURIComponent(match[1]) : null;
    },

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

    syncAuthUI() {
        const isAuthenticatedView = this.state.authenticated && this.state.currentView !== 'login';
        const processBtn = document.getElementById('btn-process');
        const logoutBtn = document.getElementById('btn-logout');
        const indicator = document.getElementById('status-indicator');

        if (processBtn) processBtn.classList.toggle('hidden', !isAuthenticatedView);
        if (logoutBtn) logoutBtn.classList.toggle('hidden', !isAuthenticatedView);
        if (indicator) indicator.classList.toggle('hidden', !isAuthenticatedView || !this.state.statusPollingInterval);
    },

    navigate(view) {
        if (view !== 'login' && !this.state.authenticated) {
            this.navigate('login');
            return;
        }

        this.state.currentView = view;
        const main = document.getElementById('main-content');
        this.syncAuthUI();

        if (view === 'login') {
            const template = document.getElementById('view-login').content.cloneNode(true);
            main.innerHTML = '';
            main.appendChild(template);
            return;
        }

        if (view === 'queue') {
            const template = document.getElementById('view-queue').content.cloneNode(true);
            main.innerHTML = '';
            main.appendChild(template);
            this.initializeExecutionOptionsUI();
            this.loadQueue();

            // Search listener
            const searchInput = document.getElementById('search');
            if (searchInput) {
                searchInput.addEventListener('input', (e) => {
                    this.renderQueue(e.target.value);
                });
            }
        } else if (view === 'details') {
            const template = document.getElementById('view-details').content.cloneNode(true);
            main.innerHTML = '';
            main.appendChild(template);
            this.renderDetails();
        }
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
                alert('Credenciais inválidas');
            }
        } catch (err) {
            console.error(err);
            alert('Erro ao realizar login');
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

    async loadQueue() {
        try {
            const res = await this.authorizedFetch(`${API_URL}/conciliations`);
            const data = await res.json();
            this.state.conciliations = data || [];
            this.renderQueue();
        } catch (err) {
            console.error(err);
            if (err.message !== 'Sessão expirada. Faça login novamente.') {
                alert('Erro ao carregar conciliações');
            }
        }
    },

    renderQueue(filter = '') {
        const list = document.getElementById('conciliation-list');
        const countSpan = document.getElementById('pending-count');
        if (!list) return;

        list.innerHTML = '';
        const search = filter.toLowerCase();

        const filtered = this.state.conciliations.filter(c =>
            c.dono.toLowerCase().includes(search) ||
            c.banco.toLowerCase().includes(search) ||
            c.valor.toString().includes(search)
        );

        countSpan.textContent = `${filtered.length} pendentes`;

        filtered.forEach(item => {
            const el = document.createElement('div');
            el.className = 'conciliation-item';
            el.innerHTML = `
                <div class="item-info">
                    <h3>${item.descricao}</h3>
                    <div class="item-meta">
                        <span class="badge">${item.dono}</span>
                        <span>${item.banco}</span>
                        <span>${item.data}</span>
                    </div>
                </div>
                <div class="item-value">
                    <div style="text-align: right; font-weight: bold; font-size: 1.2rem;">
                        R$ ${item.valor.toFixed(2)}
                    </div>
                    <div class="candidates-count" style="text-align: right; font-size: 0.9rem;">
                        ${item.candidateCount} candidatas
                    </div>
                </div>
            `;
            el.onclick = () => this.loadDetails(item.difRowIndex);
            list.appendChild(el);
        });
    },

    loadExecutionOptions() {
        const fallback = { ...DEFAULT_EXECUTION_OPTIONS };
        try {
            const raw = localStorage.getItem(EXECUTION_OPTIONS_STORAGE_KEY);
            if (!raw) return fallback;
            const parsed = JSON.parse(raw);
            return this.normalizeExecutionOptions(parsed);
        } catch (err) {
            console.warn('Falha ao carregar opções salvas, usando padrão:', err);
            return fallback;
        }
    },

    normalizeExecutionOptions(options) {
        const normalized = { ...DEFAULT_EXECUTION_OPTIONS };
        const incoming = options && typeof options === 'object' ? options : {};

        if (typeof incoming.startDate === 'string' && this.isValidISODate(incoming.startDate)) {
            normalized.startDate = incoming.startDate;
        }

        if (Array.isArray(incoming.excludeCategories)) {
            normalized.excludeCategories = incoming.excludeCategories
                .filter(v => typeof v === 'string' && v.trim() !== '')
                .map(v => v.trim());
        }

        return normalized;
    },

    saveExecutionOptions(options) {
        try {
            localStorage.setItem(EXECUTION_OPTIONS_STORAGE_KEY, JSON.stringify(options));
        } catch (err) {
            console.warn('Falha ao salvar opções de execução:', err);
        }
    },

    initializeExecutionOptionsUI() {
        this.state.executionOptions = this.loadExecutionOptions();
        this.fillExecutionOptionsForm(this.state.executionOptions);
        this.renderExecutionOptionsSummary(this.state.executionOptions);
        this.clearStartDateError();

        const startDateInput = document.getElementById('opt-start-date');
        const samePersonInput = document.getElementById('opt-exclude-same-person');
        const creditCardInput = document.getElementById('opt-exclude-credit-card');
        const resetBtn = document.getElementById('btn-options-reset');

        const onChange = () => {
            const options = this.readExecutionOptionsFromForm();
            this.state.executionOptions = options;
            this.saveExecutionOptions(options);
            this.renderExecutionOptionsSummary(options);
        };

        if (startDateInput) {
            startDateInput.addEventListener('input', () => {
                this.clearStartDateError();
                onChange();
            });
        }
        if (samePersonInput) samePersonInput.addEventListener('change', onChange);
        if (creditCardInput) creditCardInput.addEventListener('change', onChange);
        if (resetBtn) {
            resetBtn.addEventListener('click', () => {
                const defaults = { ...DEFAULT_EXECUTION_OPTIONS };
                this.state.executionOptions = defaults;
                this.fillExecutionOptionsForm(defaults);
                this.clearStartDateError();
                this.saveExecutionOptions(defaults);
                this.renderExecutionOptionsSummary(defaults);
            });
        }
    },

    fillExecutionOptionsForm(options) {
        const startDateInput = document.getElementById('opt-start-date');
        const samePersonInput = document.getElementById('opt-exclude-same-person');
        const creditCardInput = document.getElementById('opt-exclude-credit-card');

        if (startDateInput) startDateInput.value = options.startDate || '';
        if (samePersonInput) {
            samePersonInput.checked = (options.excludeCategories || []).includes('Same person transfer');
        }
        if (creditCardInput) {
            creditCardInput.checked = (options.excludeCategories || []).includes('Credit card payment');
        }
    },

    readExecutionOptionsFromForm() {
        const startDateInput = document.getElementById('opt-start-date');
        if (!startDateInput) {
            return this.normalizeExecutionOptions(this.state.executionOptions);
        }

        const startDate = startDateInput.value || '';
        const excludeCategories = [];

        const samePersonInput = document.getElementById('opt-exclude-same-person');
        const creditCardInput = document.getElementById('opt-exclude-credit-card');

        if (samePersonInput?.checked) excludeCategories.push(samePersonInput.value);
        if (creditCardInput?.checked) excludeCategories.push(creditCardInput.value);

        const options = {};
        if (startDate) options.startDate = startDate;
        options.excludeCategories = excludeCategories;

        return this.normalizeExecutionOptions(options);
    },

    renderExecutionOptionsSummary(options) {
        const summaryEl = document.getElementById('execution-options-summary');
        if (!summaryEl) return;

        const date = options.startDate ? new Date(`${options.startDate}T00:00:00`).toLocaleDateString('pt-BR') : '-';
        const excludedCount = (options.excludeCategories || []).length;
        summaryEl.textContent = `Data inicial: ${date} | ${excludedCount} categoria(s) excluída(s)`;
    },

    isValidISODate(value) {
        if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
        const date = new Date(`${value}T00:00:00Z`);
        return !Number.isNaN(date.getTime()) && date.toISOString().slice(0, 10) === value;
    },

    validateStartDateOrThrow() {
        const options = this.state.executionOptions || {};
        if (!options.startDate || !this.isValidISODate(options.startDate)) {
            this.showStartDateError('Informe uma data válida no formato YYYY-MM-DD.');
            throw new Error('Data inicial inválida');
        }

        const inputDate = new Date(`${options.startDate}T00:00:00`);
        const today = new Date();
        today.setHours(0, 0, 0, 0);
        if (inputDate > today) {
            this.showStartDateError('A data inicial não pode ser futura.');
            throw new Error('Data inicial futura');
        }
    },

    showStartDateError(message) {
        const errorEl = document.getElementById('opt-start-date-error');
        if (!errorEl) return;
        errorEl.textContent = message;
        errorEl.classList.remove('hidden');
    },

    clearStartDateError() {
        const errorEl = document.getElementById('opt-start-date-error');
        if (!errorEl) return;
        errorEl.textContent = '';
        errorEl.classList.add('hidden');
    },

    async loadDetails(id) {
        try {
            const res = await this.authorizedFetch(`${API_URL}/conciliations/${id}`);
            const data = await res.json();
            this.state.details = data;
            this.state.selectedCandidates.clear();
            this.navigate('details');
        } catch (err) {
            console.error(err);
            alert('Erro ao carregar detalhes');
        }
    },

    renderDetails() {
        if (!this.state.details) return;

        const ref = this.state.details.reference;
        const candidates = this.state.details.candidates || [];

        // Render Reference
        const refContainer = document.getElementById('ref-details');
        refContainer.innerHTML = this.renderCardContent(ref);

        // Render Candidates
        const candList = document.getElementById('candidates-list');
        candList.innerHTML = '';

        if (candidates.length === 0) {
            candList.innerHTML = '<p style="color: var(--text-muted); text-align: center;">Nenhuma candidata encontrada.</p>';
        }

        candidates.forEach(c => {
            const el = document.createElement('div');
            el.className = 'candidate-item';
            const isSelected = this.state.selectedCandidates.has(c.rowIndex);

            el.innerHTML = `
                <input type="checkbox" ${isSelected ? 'checked' : ''} onchange="app.toggleCandidate(${c.rowIndex})">
                <div class="candidate-details">
                    ${this.renderCardContent(c, true)}
                </div>
            `;
            candList.appendChild(el);
        });
    },

    renderCardContent(item, compact = false) {
        return `
            <div class="data-row">
                <span class="label">Descrição</span>
                <span>${item.descricao}</span>
            </div>
            <div class="data-row">
                <span class="label">Valor</span>
                <span style="font-weight: bold; color: ${item.sheet === 'DIF' ? 'var(--secondary)' : 'var(--success)'}">
                    R$ ${item.valor.toFixed(2)}
                </span>
            </div>
            <div class="data-row">
                <span class="label">Data</span>
                <span>${item.data}</span>
            </div>
            ${!compact ? `
            <div class="data-row">
                <span class="label">Conta</span>
                <span>${item.conta}</span>
            </div>
            <div class="data-row">
                <span class="label">ID Parcela</span>
                <span>${item.idParcela || '-'}</span>
            </div>
            ` : ''}
        `;
    },

    toggleCandidate(index) {
        if (this.state.selectedCandidates.has(index)) {
            this.state.selectedCandidates.delete(index);
        } else {
            this.state.selectedCandidates.add(index);
        }
    },

    async acceptSelection() {
        if (this.state.selectedCandidates.size === 0) {
            alert('Selecione pelo menos uma candidata.');
            return;
        }

        const difIndex = this.state.details.reference.rowIndex;
        const esIndices = Array.from(this.state.selectedCandidates);

        try {
            const res = await this.authorizedFetch(`${API_URL}/conciliations/${difIndex}/accept`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ esRowIndices: esIndices })
            });

            if (res.ok) {
                alert('Conciliação realizada com sucesso!');
                this.navigate('queue');
            } else {
                const txt = await res.text();
                alert('Erro: ' + txt);
            }
        } catch (err) {
            console.error(err);
            alert('Erro na requisição');
        }
    },

    async rejectCurrent() {
        if (!confirm('Tem certeza que deseja rejeitar esta conciliação? A referência será movida para REJ.')) return;

        const difIndex = this.state.details.reference.rowIndex;

        try {
            const res = await this.authorizedFetch(`${API_URL}/conciliations/${difIndex}/reject`, {
                method: 'POST'
            });

            if (res.ok) {
                alert('Rejeitada com sucesso!');
                this.navigate('queue');
            } else {
                const txt = await res.text();
                alert('Erro: ' + txt);
            }
        } catch (err) {
            console.error(err);
            alert('Erro na requisição');
        }
    },

    // --- Execution Logic ---

    async startTransactionProcessing() {
        if (this.state.statusPollingInterval) {
            this.openStatusModal();
            return;
        }

        try {
            this.state.executionOptions = this.readExecutionOptionsFromForm();
            this.saveExecutionOptions(this.state.executionOptions);
            this.renderExecutionOptionsSummary(this.state.executionOptions);
            this.validateStartDateOrThrow();

            const res = await this.authorizedFetch(`${EXECUTION_API_URL}/transactions`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    options: this.state.executionOptions,
                    sheet: DEFAULT_EXECUTION_PAYLOAD.sheet,
                    artifacts: DEFAULT_EXECUTION_PAYLOAD.artifacts,
                    banks: DEFAULT_EXECUTION_PAYLOAD.banks
                })
            });
            const payload = await this.parseResponseSafely(res);

            if (res.status === 202) {
                const data = payload.data;
                if (!data?.executionId) {
                    throw new Error('Resposta inválida da API de Processamento');
                }
                this.state.currentExecution = data;

                this.openStatusModal();
                this.startStatusPolling(data.executionId);
            } else {
                alert('Erro ao iniciar: ' + this.getErrorMessage(payload, `HTTP ${res.status}`));
            }
        } catch (err) {
            console.error(err);
            alert(err.message || 'Erro ao conectar com a API de Processamento');
        }
    },

    startStatusPolling(executionId) {
        if (this.state.statusPollingInterval) clearInterval(this.state.statusPollingInterval);

        this.state.statusPollingInterval = setInterval(() => {
            this.updateExecutionStatus(executionId);
        }, 2000);

        this.syncAuthUI();
        this.updateExecutionStatus(executionId);
    },

    async updateExecutionStatus(executionId) {
        try {
            const res = await this.authorizedFetch(`${EXECUTION_API_URL}/transactions/${executionId}/status`);
            if (!res.ok) throw new Error('Status not found');

            const payload = await this.parseResponseSafely(res);
            const status = payload.data;
            if (!status) throw new Error('Resposta inválida no status da execução');

            // Update UI if modal is open
            const statusBadge = document.getElementById('execution-status');
            if (statusBadge) {
                statusBadge.textContent = status.status;
                statusBadge.className = 'status-badge ' + status.status;

                document.getElementById('progress-fill').style.width = status.progress + '%';
                document.getElementById('progress-text').textContent = status.progress + '%';
                document.getElementById('current-step').textContent = this.translateStep(status.step);
            }

            // End polling if finished
            if (status.status === 'COMPLETED' || status.status === 'FAILED') {
                clearInterval(this.state.statusPollingInterval);
                this.state.statusPollingInterval = null;
                this.syncAuthUI();

                await this.loadExecutionDetails(executionId);

                if (status.status === 'COMPLETED' && this.state.currentView === 'queue') {
                    this.loadQueue();
                }
            }
        } catch (err) {
            console.error('Polling error:', err);
            clearInterval(this.state.statusPollingInterval);
            this.state.statusPollingInterval = null;
            this.syncAuthUI();
        }
    },

    async loadExecutionDetails(executionId) {
        try {
            const res = await this.authorizedFetch(`${EXECUTION_API_URL}/transactions/${executionId}`);
            const payload = await this.parseResponseSafely(res);
            const data = payload.data;
            if (!data) throw new Error('Resposta inválida nos detalhes da execução');

            this.updateMetrics(data.metrics);
            this.loadExecutionHistory();
        } catch (err) {
            console.error(err);
        }
    },

    updateMetrics(metrics) {
        if (!metrics) return;
        const m1 = document.getElementById('metric-transactions');
        const m2 = document.getElementById('metric-installments');
        const m3 = document.getElementById('metric-duplicates');

        if (m1) m1.textContent = metrics.transactionsFetched || 0;
        if (m2) m2.textContent = metrics.installmentsCreated || 0;
        if (m3) m3.textContent = metrics.duplicatesRemoved || 0;
    },

    async loadExecutionHistory() {
        try {
            const res = await this.authorizedFetch(`${EXECUTION_API_URL}/transactions`);
            const payload = await this.parseResponseSafely(res);
            const data = payload.data;
            if (!data) throw new Error('Resposta inválida no histórico de execuções');
            this.state.executionHistory = data.items || [];

            const list = document.getElementById('history-list');
            if (!list) return;

            list.innerHTML = '';
            this.state.executionHistory.slice(0, 5).forEach(item => {
                const date = new Date(item.createdAt).toLocaleString('pt-BR');
                const el = document.createElement('div');
                el.className = 'history-item';
                el.innerHTML = `
                    <div class="hist-info">
                        <span class="hist-date">${date}</span>
                        <span class="hist-meta">${this.translateStep(item.step)}</span>
                    </div>
                    <div class="hist-status ${item.status}"></div>
                `;
                list.appendChild(el);
            });
        } catch (err) {
            console.error(err);
        }
    },

    openStatusModal() {
        const container = document.getElementById('status-modal-container');
        const template = document.getElementById('modal-execution-status').content.cloneNode(true);
        container.innerHTML = '';
        container.appendChild(template);

        this.loadExecutionHistory();

        // If there's an active execution, update its status immediately in the modal
        if (this.state.currentExecution && this.state.statusPollingInterval) {
            this.updateExecutionStatus(this.state.currentExecution.executionId);
        }
    },

    closeStatusModal() {
        const modal = document.getElementById('status-modal-container');
        if (modal) modal.innerHTML = '';
    },

    translateStep(step) {
        const steps = {
            'QUEUED': 'Na fila',
            'STARTED': 'Iniciado',
            'FETCHING_TRANSACTIONS': 'Buscando transações',
            'CREATING_INSTALLMENTS': 'Criando parcelas',
            'DEDUPLICATING': 'Removendo duplicatas',
            'GENERATING_CSV': 'Gerando CSV',
            'UPDATING_SHEET': 'Atualizando planilha',
            'FINALIZING': 'Finalizando',
            'DONE': 'Concluído',
            'ERROR': 'Erro'
        };
        return steps[step] || step;
    }
};

// Start
app.init();
