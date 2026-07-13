import { API_URL, EXECUTION_OPTIONS_STORAGE_KEY, DEFAULT_EXECUTION_OPTIONS } from './constants.js';

export const queueModule = {
    async loadQueue() {
        try {
            const [conciliationsRes, nonRecurringRes] = await Promise.all([
                this.authorizedFetch(`${API_URL}/conciliations`),
                this.authorizedFetch(`${API_URL}/dif/non-recurring`)
            ]);

            const conciliations = await conciliationsRes.json();
            const nonRecurring = await nonRecurringRes.json();
            this.state.conciliations = conciliations || [];
            this.state.nonRecurringDif = nonRecurring || [];
            this.state.pendingCategoryEdits = {};
            this.state.pendingDateEdits = {};
            this.renderQueue();
        } catch (err) {
            console.error(err);
            if (err.message !== 'Sessão expirada. Faça login novamente.') {
                this.showNotification('Erro ao carregar conciliações', 'error');
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

        this.renderNonRecurringList(search);
    },

    renderNonRecurringList(search = '') {
        const panel = document.getElementById('non-recurring-panel');
        const list = document.getElementById('non-recurring-list');
        const count = document.getElementById('non-recurring-count');
        const copyAllBtn = document.getElementById('btn-copy-all-non-recurring');
        if (!panel || !list || !count || !copyAllBtn) return;

        const items = (this.state.nonRecurringDif || []).filter(item => {
            if (!search) return true;
            return item.dono.toLowerCase().includes(search) ||
                item.banco.toLowerCase().includes(search) ||
                item.valor.toString().includes(search) ||
                item.descricao.toLowerCase().includes(search);
        });

        count.textContent = `${items.length} itens`;
        list.innerHTML = '';
        panel.classList.toggle('hidden', items.length === 0);
        copyAllBtn.disabled = items.length === 0;

        items.forEach(item => {
            const row = document.createElement('article');
            row.className = 'non-recurring-item';
            const categoriaAtual = this.getCategoryDraft(item);
            const hasUnsavedCategory = categoriaAtual !== (item.categoria || '');
            const dataAtual = this.getDateDraft(item);
            const hasUnsavedDate = dataAtual !== (item.data || '');

            row.innerHTML = `
                <div class="non-recurring-main">
                    <div class="non-recurring-title">${this.escapeHtml(item.descricao || '-')}</div>
                    <div class="non-recurring-meta">
                        <span>${this.escapeHtml(item.dono || '-')}</span>
                        <span>${this.escapeHtml(item.banco || '-')}</span>
                        <span>${this.escapeHtml(item.data || '-')}</span>
                        <span>R$ ${(item.valor || 0).toFixed(2)}</span>
                    </div>
                </div>
                <div class="non-recurring-category">
                    <label for="cat-${item.difRowIndex}">
                        Categoria
                        <span class="unsaved-indicator ${hasUnsavedCategory ? '' : 'hidden'}">não salvo</span>
                    </label>
                    <input id="cat-${item.difRowIndex}" type="text" class="glass-input non-recurring-category-input" value="${this.escapeHtml(categoriaAtual)}">
                    <button type="button" class="btn-ghost" data-action="save-category" data-id="${item.difRowIndex}" ${hasUnsavedCategory ? '' : 'disabled'}>Salvar categoria</button>
                </div>
                <div class="non-recurring-date">
                    <label for="date-${item.difRowIndex}">
                        Data
                        <span class="unsaved-indicator date-unsaved-indicator ${hasUnsavedDate ? '' : 'hidden'}">não salvo</span>
                    </label>
                    <input id="date-${item.difRowIndex}" type="date" class="glass-input" value="${this.escapeHtml(dataAtual)}">
                    <button type="button" class="btn-ghost" data-action="save-date" data-id="${item.difRowIndex}" ${hasUnsavedDate ? '' : 'disabled'}>Salvar data</button>
                </div>
                <div class="non-recurring-row-actions">
                    <button type="button" class="btn-accept" data-action="move-es" data-id="${item.difRowIndex}">Copiar</button>
                    <button type="button" class="btn-reject" data-action="move-rej" data-id="${item.difRowIndex}">Rejeitar</button>
                </div>
            `;

            const categoryInput = row.querySelector(`#cat-${item.difRowIndex}`);
            const unsavedIndicator = row.querySelector('.unsaved-indicator');
            const saveCategoryButton = row.querySelector('button[data-action="save-category"]');
            if (categoryInput) {
                categoryInput.addEventListener('input', (e) => {
                    const nextValue = e.target.value;
                    this.state.pendingCategoryEdits[item.difRowIndex] = nextValue;
                    const changed = nextValue !== (item.categoria || '');
                    if (unsavedIndicator) unsavedIndicator.classList.toggle('hidden', !changed);
                    if (saveCategoryButton) saveCategoryButton.disabled = !changed;
                });
            }
            if (saveCategoryButton) {
                saveCategoryButton.addEventListener('click', () => this.saveNonRecurringCategory(item.difRowIndex));
            }

            const dateInput = row.querySelector(`#date-${item.difRowIndex}`);
            const dateUnsavedIndicator = row.querySelector('.date-unsaved-indicator');
            const saveDateButton = row.querySelector('button[data-action="save-date"]');
            if (dateInput) {
                dateInput.addEventListener('input', (e) => {
                    const nextValue = e.target.value;
                    this.state.pendingDateEdits[item.difRowIndex] = nextValue;
                    const changed = nextValue !== (item.data || '');
                    if (dateUnsavedIndicator) dateUnsavedIndicator.classList.toggle('hidden', !changed);
                    if (saveDateButton) saveDateButton.disabled = !changed;
                });
            }
            if (saveDateButton) {
                saveDateButton.addEventListener('click', () => this.saveNonRecurringDate(item.difRowIndex));
            }

            const moveEsBtn = row.querySelector('button[data-action="move-es"]');
            if (moveEsBtn) {
                moveEsBtn.addEventListener('click', () => this.copyNonRecurringToES(item.difRowIndex));
            }

            const moveRejBtn = row.querySelector('button[data-action="move-rej"]');
            if (moveRejBtn) {
                moveRejBtn.addEventListener('click', () => this.rejectNonRecurringToREJ(item.difRowIndex));
            }

            list.appendChild(row);
        });
    },

    getCategoryDraft(item) {
        const draft = this.state.pendingCategoryEdits[item.difRowIndex];
        if (typeof draft === 'string') return draft;
        return item.categoria || '';
    },

    getDateDraft(item) {
        const draft = this.state.pendingDateEdits[item.difRowIndex];
        if (typeof draft === 'string') return draft;
        return item.data || '';
    },

    // Edições de categoria/data são endereçadas pelo IdParcela (identidade estável da
    // transação), não pelo índice da linha da DIF — ver docs/adr/0004. O idParcela já
    // vem no NonRecurringDifSummary, então nenhuma busca nova é necessária.
    findNonRecurringItem(difRowIndex) {
        return this.state.nonRecurringDif.find(item => item.difRowIndex === difRowIndex);
    },

    async saveNonRecurringCategory(difRowIndex) {
        const categoria = this.state.pendingCategoryEdits[difRowIndex];
        if (typeof categoria !== 'string') return;

        const item = this.findNonRecurringItem(difRowIndex);
        if (!item || !item.idParcela) {
            alert('Transação sem identificador — recarregue a lista antes de salvar.');
            return;
        }

        try {
            const res = await this.authorizedFetch(`${API_URL}/dif/non-recurring/category`, {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ idParcela: item.idParcela, categoria })
            });

            if (res.status === 404) {
                // A transação não está mais na HOM (reimport raro). Avisa e preserva os
                // rascunhos das outras linhas — a lista NÃO recarrega sozinha.
                alert('Essa transação não está mais na homologação — recarregue a lista.');
                return;
            }
            if (!res.ok) {
                const txt = await res.text();
                throw new Error(txt || 'Falha ao salvar categoria');
            }

            this.state.nonRecurringDif = this.state.nonRecurringDif.map(it =>
                it.difRowIndex === difRowIndex ? { ...it, categoria } : it
            );
            delete this.state.pendingCategoryEdits[difRowIndex];
            this.renderQueue(document.getElementById('search')?.value || '');
        } catch (err) {
            console.error(err);
            alert(`Erro ao salvar categoria: ${err.message}`);
        }
    },

    async saveNonRecurringDate(difRowIndex) {
        const data = this.state.pendingDateEdits[difRowIndex];
        if (typeof data !== 'string') return;

        const item = this.findNonRecurringItem(difRowIndex);
        if (!item || !item.idParcela) {
            alert('Transação sem identificador — recarregue a lista antes de salvar.');
            return;
        }

        try {
            const res = await this.authorizedFetch(`${API_URL}/dif/non-recurring/date`, {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ idParcela: item.idParcela, data })
            });

            if (res.status === 404) {
                alert('Essa transação não está mais na homologação — recarregue a lista.');
                return;
            }
            if (!res.ok) {
                const txt = await res.text();
                throw new Error(txt || 'Falha ao salvar data');
            }

            this.state.nonRecurringDif = this.state.nonRecurringDif.map(it =>
                it.difRowIndex === difRowIndex ? { ...it, data } : it
            );
            delete this.state.pendingDateEdits[difRowIndex];
            this.renderQueue(document.getElementById('search')?.value || '');
        } catch (err) {
            console.error(err);
            alert(`Erro ao salvar data: ${err.message}`);
        }
    },

    async copyNonRecurringToES(difRowIndex) {
        try {
            const res = await this.authorizedFetch(`${API_URL}/dif/non-recurring/${difRowIndex}/move-to-es`, {
                method: 'POST'
            });
            if (!res.ok) {
                const txt = await res.text();
                throw new Error(txt || 'Falha ao copiar para ES');
            }

            await this.loadQueue();
        } catch (err) {
            console.error(err);
            alert(`Erro ao copiar para ES: ${err.message}`);
        }
    },

    async rejectNonRecurringToREJ(difRowIndex) {
        if (!confirm('Tem certeza que deseja rejeitar esta transação?')) return;

        try {
            const res = await this.authorizedFetch(`${API_URL}/dif/non-recurring/${difRowIndex}/move-to-rej`, {
                method: 'POST'
            });
            if (!res.ok) {
                const txt = await res.text();
                throw new Error(txt || 'Falha ao mover para REJ');
            }

            await this.loadQueue();
        } catch (err) {
            console.error(err);
            alert(`Erro ao mover para REJ: ${err.message}`);
        }
    },

    async copyAllNonRecurringToES() {
        const total = (this.state.nonRecurringDif || []).length;
        if (total === 0) return;
        if (!confirm(`Copiar ${total} linha(s) de DIF não recorrente para ES?`)) return;

        try {
            const res = await this.authorizedFetch(`${API_URL}/dif/non-recurring/move-all-to-es`, {
                method: 'POST'
            });
            if (!res.ok) {
                const txt = await res.text();
                throw new Error(txt || 'Falha ao copiar todas para ES');
            }

            const payload = await this.parseResponseSafely(res);
            const moved = payload.data?.movedToES ?? total;
            alert(`${moved} linha(s) copiada(s) para ES.`);
            await this.loadQueue();
        } catch (err) {
            console.error(err);
            alert(`Erro ao copiar todas para ES: ${err.message}`);
        }
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
};
