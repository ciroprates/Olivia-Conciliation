import { EXECUTION_API_URL, DEFAULT_EXECUTION_PAYLOAD } from './constants.js';

export const executionModule = {
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
                this.showNotification('Erro ao iniciar: ' + this.getErrorMessage(payload, `HTTP ${res.status}`), 'error');
            }
        } catch (err) {
            console.error(err);
            this.showNotification(err.message || 'Erro ao conectar com a API de Processamento', 'error');
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

            const statusBadge = document.getElementById('execution-status');
            if (statusBadge) {
                statusBadge.textContent = status.status;
                statusBadge.className = 'status-badge ' + status.status;

                document.getElementById('progress-fill').style.width = status.progress + '%';
                document.getElementById('progress-text').textContent = status.progress + '%';
                document.getElementById('current-step').textContent = this.translateStep(status.step);
            }

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
    },
};
