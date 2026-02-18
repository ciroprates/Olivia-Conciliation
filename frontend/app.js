const API_URL = 'https://bff.olivinha.site/api';
const EXECUTION_API_URL = 'https://api.olivinha.site';

const app = {
    state: {
        currentView: 'queue',
        conciliations: [],
        details: null,
        selectedCandidates: new Set(),
        currentExecution: null,
        executionHistory: [],
        statusPollingInterval: null,
        token: localStorage.getItem('olivia_auth_token')
    },

    init() {
        if (!this.state.token) {
            this.navigate('login');
        } else {
            this.navigate('queue');
        }
    },

    async authorizedFetch(url, options = {}) {
        const headers = options.headers || {};
        if (this.state.token) {
            headers['Authorization'] = `Bearer ${this.state.token}`;
        }

        const res = await fetch(url, { ...options, headers });

        if (res.status === 401) {
            this.logout();
            throw new Error('Sessão expirada. Faça login novamente.');
        }

        return res;
    },

    navigate(view) {
        this.state.currentView = view;
        const main = document.getElementById('main-content');
        const logoutBtn = document.getElementById('btn-logout');

        if (view === 'login') {
            const template = document.getElementById('view-login').content.cloneNode(true);
            main.innerHTML = '';
            main.appendChild(template);
            if (logoutBtn) logoutBtn.classList.add('hidden');
            return;
        }

        if (logoutBtn) logoutBtn.classList.remove('hidden');

        if (view === 'queue') {
            const template = document.getElementById('view-queue').content.cloneNode(true);
            main.innerHTML = '';
            main.appendChild(template);
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
                body: JSON.stringify({ username: user, password: pass })
            });

            if (res.ok) {
                const data = await res.json();
                this.state.token = data.token;
                localStorage.setItem('olivia_auth_token', data.token);
                this.navigate('queue');
            } else {
                alert('Credenciais inválidas');
            }
        } catch (err) {
            console.error(err);
            alert('Erro ao realizar login');
        }
    },

    logout() {
        this.state.token = null;
        localStorage.removeItem('olivia_auth_token');
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
            const res = await this.authorizedFetch(`${EXECUTION_API_URL}/v1/executions/transactions`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({})
            });

            if (res.status === 202) {
                const data = await res.json();
                this.state.currentExecution = data;

                // Show indicator
                const indicator = document.getElementById('status-indicator');
                if (indicator) indicator.classList.remove('hidden');

                this.openStatusModal();
                this.startStatusPolling(data.executionId);
            } else {
                const error = await res.json();
                alert('Erro ao iniciar: ' + (error.message || 'Erro desconhecido'));
            }
        } catch (err) {
            console.error(err);
            alert('Erro ao conectar com a API de Processamento');
        }
    },

    startStatusPolling(executionId) {
        if (this.state.statusPollingInterval) clearInterval(this.state.statusPollingInterval);

        this.state.statusPollingInterval = setInterval(() => {
            this.updateExecutionStatus(executionId);
        }, 2000);

        this.updateExecutionStatus(executionId);
    },

    async updateExecutionStatus(executionId) {
        try {
            const res = await this.authorizedFetch(`${EXECUTION_API_URL}/v1/executions/transactions/${executionId}/status`);
            if (!res.ok) throw new Error('Status not found');

            const status = await res.json();

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

                const indicator = document.getElementById('status-indicator');
                if (indicator) indicator.classList.add('hidden');

                await this.loadExecutionDetails(executionId);

                if (status.status === 'COMPLETED' && this.state.currentView === 'queue') {
                    this.loadQueue();
                }
            }
        } catch (err) {
            console.error('Polling error:', err);
            clearInterval(this.state.statusPollingInterval);
            this.state.statusPollingInterval = null;
        }
    },

    async loadExecutionDetails(executionId) {
        try {
            const res = await this.authorizedFetch(`${EXECUTION_API_URL}/v1/executions/transactions/${executionId}`);
            const data = await res.json();

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
            const res = await this.authorizedFetch(`${EXECUTION_API_URL}/v1/executions/transactions`);
            const data = await res.json();
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
