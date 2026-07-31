import { API_URL, EXECUTION_OPTIONS_STORAGE_KEY, DEFAULT_EXECUTION_OPTIONS } from './constants.js';

export const queueModule = {
    async loadQueue() {
        try {
            const [conciliationsRes, nonRecurringRes] = await Promise.all([
                this.authorizedFetch(`${API_URL}/conciliations`),
                this.authorizedFetch(`${API_URL}/dif/non-recurring`)
            ]);

            const summaries = (await conciliationsRes.json()) || [];
            const nonRecurring = (await nonRecurringRes.json()) || [];

            // GET /api/conciliations traz só o resumo (com candidateCount), sem a
            // candidata nem a categoria da referência. O layout de abas exibe a
            // melhor candidata inline e agrupa por categoria, então enriquecemos
            // cada linha com GET /conciliations/{i}. São poucas pendências por vez,
            // então buscamos os detalhes em paralelo — o N+1 é aceitável aqui.
            const enriched = await Promise.all(summaries.map(async (s) => {
                try {
                    const res = await this.authorizedFetch(`${API_URL}/conciliations/${s.difRowIndex}`);
                    const detail = await res.json();
                    const ref = detail?.reference || {};
                    const candidate = (detail?.candidates && detail.candidates[0]) || null;
                    return { ...s, categoria: ref.categoria || 'Outros', candidate };
                } catch {
                    return { ...s, categoria: 'Outros', candidate: null };
                }
            }));

            this.state.conciliations = enriched;
            this.state.nonRecurringDif = nonRecurring;
            this.state.pendingCategoryEdits = {};
            this.state.pendingDateEdits = {};
            this.pruneSelections();
            this.renderQueue();
        } catch (err) {
            console.error(err);
            if (err.message !== 'Sessão expirada. Faça login novamente.') {
                this.showNotification('Erro ao carregar conciliações', 'error');
            }
        }
    },

    // Remove das seleções em lote os índices que não existem mais após um reload.
    pruneSelections() {
        const concIds = new Set(this.state.conciliations.map(c => c.difRowIndex));
        this.state.concSelected = new Set([...this.state.concSelected].filter(id => concIds.has(id)));
        const impIds = new Set(this.state.nonRecurringDif.map(c => c.difRowIndex));
        this.state.impSelected = new Set([...this.state.impSelected].filter(id => impIds.has(id)));
    },

    renderQueue() {
        this.renderConc();
        this.renderImport();
        const cntConc = document.getElementById('cnt-conc');
        if (cntConc) cntConc.textContent = this.state.conciliations.length;
        const cntImp = document.getElementById('cnt-imp');
        if (cntImp) cntImp.textContent = this.state.nonRecurringDif.length;
    },

    // ── Aba: navegação e popover de importação ──
    switchTab(tab) {
        this.state.activeTab = tab;
        const isConc = tab === 'conc';
        document.getElementById('tab-btn-conc')?.classList.toggle('active', isConc);
        document.getElementById('tab-btn-imp')?.classList.toggle('active', !isConc);
        document.getElementById('tab-conc')?.classList.toggle('active', isConc);
        document.getElementById('tab-imp')?.classList.toggle('active', !isConc);
        const batchBar = document.getElementById('batch-bar');
        if (batchBar) batchBar.style.display = isConc ? 'flex' : 'none';
        const impBar = document.getElementById('imp-action-bar');
        if (impBar) impBar.classList.toggle('off', isConc || this.state.impSelected.size === 0);
    },

    toggleImportPopover() {
        this.state.importPopoverOpen = !this.state.importPopoverOpen;
        document.getElementById('import-popover')?.classList.toggle('open', this.state.importPopoverOpen);
    },

    closeImportPopover() {
        this.state.importPopoverOpen = false;
        document.getElementById('import-popover')?.classList.remove('open');
    },

    // ════════════════════════════════════════════
    // Aba Conciliação — tabela agrupada por categoria
    // ════════════════════════════════════════════
    renderConc() {
        const body = document.getElementById('conc-body');
        if (!body) return;

        const items = this.state.conciliations;
        if (items.length === 0) {
            body.innerHTML = '<p class="empty-state">Nenhuma parcela pendente de conciliação. 🎉</p>';
            this.updateBatchBar();
            return;
        }

        const cats = [...new Set(items.map(r => r.categoria))].sort();
        let rows = '';
        for (const cat of cats) {
            const color = this.categoryColor(cat);
            rows += `<tr class="tbl-cat-hdr"><td colspan="6"><div class="cat-hdr-content">
                <span class="cat-hdr-dot" style="background:${color}"></span>${this.escapeHtml(cat)}
                <div class="cat-hdr-line"></div></div></td></tr>`;
            for (const r of items.filter(x => x.categoria === cat)) rows += this.renderConcRow(r);
        }

        body.innerHTML = `<table class="tbl">
            <thead class="tbl-head"><tr>
                <th style="width:32px"></th>
                <th>Transação DIF</th>
                <th style="width:28px"></th>
                <th>Melhor candidata ES</th>
                <th style="text-align:right;white-space:nowrap">Valor / Δ</th>
                <th>Ações</th>
            </tr></thead>
            <tbody>${rows}</tbody>
        </table>`;
        this.updateBatchBar();
    },

    renderConcRow(r) {
        const isSel = this.state.concSelected.has(r.difRowIndex);
        const c = r.candidate;
        const difTxId = this.formatTxId('DIF', r.difRowIndex);
        const parc = this.parcelaSuffix(r.idParcela);
        const count = r.candidateCount || 0;

        const esBlock = c ? `
            <div class="tx-meta" style="margin-bottom:.22rem">
                <span class="tx-id">${this.formatTxId('ES', c.rowIndex)}</span>
                <span class="parc-badge parc-full">${count} candidata${count !== 1 ? 's' : ''}</span>
            </div>
            <div class="tx-name">${this.escapeHtml(c.descricao)}</div>
            <div class="tx-meta" style="margin-top:.18rem">
                <span class="date-badge">📅 ${this.escapeHtml(c.data)}</span>
                <span class="muted">${this.escapeHtml(c.dono)} · ${this.escapeHtml(c.banco)}</span>
            </div>` : '<span class="no-cand">Sem candidata encontrada</span>';

        return `<tr data-cid="${r.difRowIndex}" class="${isSel ? 'row-selected' : ''}">
            <td><div class="chk${isSel ? ' on' : ''}" onclick="app.toggleConcItem(${r.difRowIndex})"></div></td>
            <td>
                <div class="tx-meta" style="margin-bottom:.22rem">
                    <span class="tx-id">${difTxId}</span>
                    ${parc ? `<span class="parc-badge parc-partial">${this.escapeHtml(parc)}</span>` : ''}
                </div>
                <div class="tx-name">${this.escapeHtml(r.descricao)}</div>
                <div class="tx-meta" style="margin-top:.18rem">
                    <span class="date-badge">📅 ${this.escapeHtml(r.data)}</span>
                    <span class="muted">${this.escapeHtml(r.dono)} · ${this.escapeHtml(r.banco)}</span>
                </div>
            </td>
            <td class="arrow-col">→</td>
            <td>${esBlock}</td>
            <td style="text-align:right;white-space:nowrap">
                <div class="val-sm">${this.formatCurrency(r.valor)}</div>
                ${c ? `<div style="margin-top:.22rem">${this.deltaHtml(r.valor, c.valor)}</div>` : ''}
            </td>
            <td>
                <div class="tbl-actions">
                    ${c ? `<button class="btn btn-concil" onclick="app.acceptConciliation(${r.difRowIndex}, ${c.rowIndex})">Conciliar</button>` : ''}
                    <button class="btn btn-reject" onclick="app.rejectConciliation(${r.difRowIndex})">Rejeitar</button>
                    <button class="edit-btn" id="cebtn-${r.difRowIndex}" onclick="app.toggleConcEdit(${r.difRowIndex})">✏ cat.</button>
                </div>
            </td>
        </tr>
        <tr class="cat-edit-row" id="cedit-${r.difRowIndex}">
            <td colspan="6">
                <div class="cat-edit-inner">
                    <span class="muted" style="white-space:nowrap;font-size:.76rem">Categoria de <strong style="color:var(--text-main)">${difTxId}</strong></span>
                    <input class="cat-inp" id="ccat-${r.difRowIndex}" value="${this.escapeHtml(r.categoria)}" style="max-width:220px">
                    <button class="btn save-cat-btn" onclick="app.saveConciliationCategory(${r.difRowIndex})">Salvar</button>
                    <button class="btn btn-ghost" onclick="app.toggleConcEdit(${r.difRowIndex})">Cancelar</button>
                </div>
            </td>
        </tr>`;
    },

    toggleConcItem(difRowIndex) {
        const set = this.state.concSelected;
        if (set.has(difRowIndex)) set.delete(difRowIndex); else set.add(difRowIndex);
        this.renderConc();
    },

    toggleConcAll() {
        const ids = this.state.conciliations.map(r => r.difRowIndex);
        const allOn = ids.length > 0 && ids.every(id => this.state.concSelected.has(id));
        this.state.concSelected = new Set(allOn ? [] : ids);
        this.renderConc();
    },

    updateBatchBar() {
        const n = this.state.concSelected.size;
        const total = this.state.conciliations.length;
        const selCount = document.getElementById('conc-sel-count');
        if (selCount) selCount.textContent = n;
        const totalEl = document.getElementById('conc-total');
        if (totalEl) totalEl.textContent = total;
        const allChk = document.getElementById('batch-chk-all');
        if (allChk) {
            allChk.classList.toggle('on', n > 0 && n === total);
            allChk.style.opacity = n > 0 && n < total ? '0.6' : '1';
        }
        const cLote = document.getElementById('btn-concil-lote');
        if (cLote) cLote.disabled = n === 0;
        const rLote = document.getElementById('btn-reject-lote');
        if (rLote) rLote.disabled = n === 0;
    },

    toggleConcEdit(difRowIndex) {
        const row = document.getElementById(`cedit-${difRowIndex}`);
        const btn = document.getElementById(`cebtn-${difRowIndex}`);
        if (!row) return;
        const open = row.classList.toggle('open');
        if (btn) btn.classList.toggle('open', open);
    },

    // Edição de categoria endereçada por IdParcela (ver docs/adr/0004): escreve na
    // HOM, e a fórmula recalcula a DIF. Rota real: PATCH /dif/non-recurring/category
    // — a rota `/{difRowIndex}/category` citada na issue #6 é anterior ao ADR 0004.
    async saveConciliationCategory(difRowIndex) {
        const input = document.getElementById(`ccat-${difRowIndex}`);
        const item = this.state.conciliations.find(c => c.difRowIndex === difRowIndex);
        if (!input || !item) return;
        if (!item.idParcela) {
            this.showNotification('Transação sem identificador — recarregue a lista.', 'error');
            return;
        }
        const value = input.value;
        try {
            const res = await this.authorizedFetch(`${API_URL}/dif/non-recurring/category`, {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ idParcela: item.idParcela, categoria: value })
            });
            if (res.status === 404) {
                this.showNotification('Transação não está mais na homologação — recarregue.', 'error');
                return;
            }
            if (!res.ok) {
                const txt = await res.text();
                throw new Error(txt || 'Falha ao salvar categoria');
            }
            item.categoria = value;
            this.renderConc();
            this.showNotification(`Categoria atualizada para "${value}".`, 'success');
        } catch (err) {
            console.error(err);
            this.showNotification(`Erro ao salvar categoria: ${err.message}`, 'error');
        }
    },

    async acceptSelectedConciliations() {
        const items = [...this.state.concSelected]
            .map(id => this.state.conciliations.find(c => c.difRowIndex === id))
            .filter(c => c && c.candidate);
        if (items.length === 0) {
            this.showNotification('Nenhuma das selecionadas tem candidata para conciliar.', 'error');
            return;
        }
        if (!confirm(`Conciliar ${items.length} parcela(s) com a melhor candidata?`)) return;

        let ok = 0, fail = 0;
        for (const c of items) {
            try {
                const res = await this.authorizedFetch(`${API_URL}/conciliations/${c.difRowIndex}/accept`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ esRowIndices: [c.candidate.rowIndex] })
                });
                if (res.ok) ok++; else fail++;
            } catch { fail++; }
        }
        this.state.concSelected = new Set();
        this.showNotification(`${ok} conciliada(s)${fail ? `, ${fail} com erro` : ''}.`, fail ? 'error' : 'success');
        await this.loadQueue();
    },

    async rejectSelectedConciliations() {
        const ids = [...this.state.concSelected];
        if (ids.length === 0) return;
        if (!confirm(`Rejeitar ${ids.length} parcela(s)? Serão movidas para REJ.`)) return;

        let ok = 0, fail = 0;
        for (const id of ids) {
            try {
                const res = await this.authorizedFetch(`${API_URL}/conciliations/${id}/reject`, { method: 'POST' });
                if (res.ok) ok++; else fail++;
            } catch { fail++; }
        }
        this.state.concSelected = new Set();
        this.showNotification(`${ok} rejeitada(s)${fail ? `, ${fail} com erro` : ''}.`, fail ? 'error' : 'success');
        await this.loadQueue();
    },

    // ════════════════════════════════════════════
    // Aba Importação — cards REC (idParcela) + NRC (sem idParcela)
    // ════════════════════════════════════════════
    renderImport() {
        const body = document.getElementById('imp-body');
        if (!body) return;

        const all = this.state.nonRecurringDif || [];
        if (all.length === 0) {
            body.innerHTML = '<p class="empty-state">Nenhuma transação para importar. 🎉</p>';
            this.updateImportActionBar();
            return;
        }

        // REC = recorrente para cópia (tem idParcela, é a 1ª do conjunto);
        // NRC = não-recorrente (idParcela vazio). Ambos vêm do mesmo endpoint.
        const rec = all.filter(r => r.idParcela);
        const nrc = all.filter(r => !r.idParcela);

        const allIds = all.map(r => r.difRowIndex);
        const allOn = allIds.length > 0 && allIds.every(id => this.state.impSelected.has(id));

        body.innerHTML = `
            <div class="sel-all-row">
                <div class="chk${allOn ? ' on' : ''}" onclick="app.toggleImportAll()"></div>
                <span class="sel-label">Selecionar todas · <strong>${allIds.length} transações</strong></span>
                <button class="btn btn-copy-sel" style="margin-left:auto" onclick="app.copyAllNonRecurringToES()">Copiar todas para ES</button>
            </div>
            ${this.renderImportSection('REC', 'Parcelas completas (incluindo 1ª)', rec, true)}
            ${this.renderImportSection('N.REC', 'Não recorrentes', nrc, false)}`;
        this.updateImportActionBar();
    },

    renderImportSection(pill, title, arr, isRec) {
        if (arr.length === 0) return '';
        const pillCls = isRec ? 'pill-copy' : 'pill-nonr';
        const cats = [...new Set(arr.map(r => r.categoria || 'Outros'))].sort();
        let sections = '';
        for (const cat of cats) {
            const color = this.categoryColor(cat);
            let cards = '';
            for (const r of arr.filter(x => (x.categoria || 'Outros') === cat)) cards += this.renderImportCard(r, isRec);
            sections += `<div class="imp-cat-hdr"><span class="imp-cat-dot" style="background:${color}"></span>${this.escapeHtml(cat)}<div class="imp-sub-hdr-line"></div></div><div class="imp-list">${cards}</div>`;
        }
        return `<div class="imp-sub-hdr"><span class="pill ${pillCls}">${pill}</span>${title}<div class="imp-sub-hdr-line"></div><span class="pill pill-gray">${arr.length}</span></div>${sections}`;
    },

    renderImportCard(r, isRec) {
        const isSel = this.state.impSelected.has(r.difRowIndex);
        const txId = this.formatTxId(isRec ? 'REC' : 'NRC', r.difRowIndex);
        const parc = this.parcelaSuffix(r.idParcela);
        const badge = isRec
            ? `<span class="parc-badge parc-full">${this.escapeHtml(parc || 'parcela')}</span>`
            : `<span class="parc-badge parc-nonr">${this.escapeHtml(r.data || '')}</span>`;
        // Categoria/data são editadas por IdParcela (ADR 0004); sem id estável (NRC),
        // não há como endereçar a HOM, então a edição inline só aparece para REC.
        const canEdit = !!r.idParcela;
        const catValue = this.getCategoryDraft(r);
        const dateValue = this.getDateDraft(r);

        return `<div class="imp-card${isSel ? ' selected' : ''}" data-id="${r.difRowIndex}">
            <div class="imp-main">
                <div class="chk${isSel ? ' on' : ''}" onclick="app.toggleImportItem(${r.difRowIndex})"></div>
                <div class="imp-info">
                    <div class="imp-meta" style="margin-bottom:.2rem">
                        <span class="tx-id">${txId}</span>
                        ${badge}
                    </div>
                    <div class="imp-name">${this.escapeHtml(r.descricao || '-')}</div>
                    <div class="imp-meta" style="margin-top:.18rem">
                        <span class="date-badge">📅 ${this.escapeHtml(r.data || '-')}</span>
                        <span class="muted">${this.escapeHtml(r.dono || '-')} · ${this.escapeHtml(r.banco || '-')}</span>
                    </div>
                </div>
                <div class="imp-right">
                    <span class="imp-val">${this.formatCurrency(r.valor)}</span>
                    <div class="imp-btns">
                        ${canEdit ? `<button class="edit-btn" id="imp-eb-${r.difRowIndex}" onclick="app.toggleImportEdit(${r.difRowIndex})">✏ cat.</button>` : ''}
                        <button class="btn btn-accept" onclick="app.copyNonRecurringToES(${r.difRowIndex})">Copiar</button>
                        <button class="btn btn-reject" onclick="app.rejectNonRecurringToREJ(${r.difRowIndex})">Rej.</button>
                    </div>
                </div>
            </div>
            ${canEdit ? `
            <div class="imp-expand" id="imp-exp-${r.difRowIndex}">
                <div class="imp-expand-inner">
                    <input class="cat-inp" id="cat-${r.difRowIndex}" value="${this.escapeHtml(catValue)}" placeholder="Categoria">
                    <button class="btn save-cat-btn" onclick="app.saveNonRecurringCategory(${r.difRowIndex})">Salvar cat.</button>
                    <input class="cat-inp" type="date" id="date-${r.difRowIndex}" value="${this.escapeHtml(dateValue)}" style="max-width:150px">
                    <button class="btn save-cat-btn" onclick="app.saveNonRecurringDate(${r.difRowIndex})">Salvar data</button>
                </div>
            </div>` : ''}
        </div>`;
    },

    toggleImportItem(difRowIndex) {
        const set = this.state.impSelected;
        if (set.has(difRowIndex)) set.delete(difRowIndex); else set.add(difRowIndex);
        const card = document.querySelector(`.imp-card[data-id="${difRowIndex}"]`);
        if (card) {
            const on = set.has(difRowIndex);
            card.classList.toggle('selected', on);
            card.querySelector('.chk')?.classList.toggle('on', on);
        }
        this.updateImportActionBar();
    },

    toggleImportAll() {
        const ids = (this.state.nonRecurringDif || []).map(r => r.difRowIndex);
        const allOn = ids.length > 0 && ids.every(id => this.state.impSelected.has(id));
        this.state.impSelected = new Set(allOn ? [] : ids);
        this.renderImport();
    },

    toggleImportEdit(difRowIndex) {
        const exp = document.getElementById(`imp-exp-${difRowIndex}`);
        const btn = document.getElementById(`imp-eb-${difRowIndex}`);
        if (!exp) return;
        const open = exp.classList.toggle('open');
        if (btn) btn.classList.toggle('open', open);
    },

    updateImportActionBar() {
        const n = this.state.impSelected.size;
        const bar = document.getElementById('imp-action-bar');
        if (bar) bar.classList.toggle('off', n === 0 || this.state.activeTab !== 'imp');
        const lbl = document.getElementById('imp-sel-lbl');
        if (lbl) lbl.textContent = `${n} selecionada${n !== 1 ? 's' : ''}`;
        const btn = document.getElementById('imp-copy-btn');
        if (btn) btn.textContent = `Copiar ${n}`;
    },

    clearImportSelection() {
        this.state.impSelected = new Set();
        this.renderImport();
    },

    async copySelectedImport() {
        const ids = [...this.state.impSelected];
        if (ids.length === 0) return;
        if (!confirm(`Copiar ${ids.length} transação(ões) para ES?`)) return;

        let ok = 0, fail = 0;
        for (const id of ids) {
            try {
                const res = await this.authorizedFetch(`${API_URL}/dif/non-recurring/${id}/move-to-es`, { method: 'POST' });
                if (res.ok) ok++; else fail++;
            } catch { fail++; }
        }
        this.state.impSelected = new Set();
        this.showNotification(`${ok} copiada(s) para ES${fail ? `, ${fail} com erro` : ''}.`, fail ? 'error' : 'success');
        await this.loadQueue();
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

    // saveNonRecurringField salva um campo editável (categoria/data) endereçado por
    // IdParcela — ver docs/adr/0004. `route` é o sufixo da rota (category/date);
    // `field` é a chave em pt-BR (categoria/data); `inputId` é o prefixo do id do
    // input no DOM (cat/date). Em 404, avisa e não recarrega (preserva o resto).
    async saveNonRecurringField(difRowIndex, { route, field, label, inputId }) {
        const input = document.getElementById(`${inputId}-${difRowIndex}`);
        if (!input) return;
        const value = input.value;

        const item = this.findNonRecurringItem(difRowIndex);
        if (!item || !item.idParcela) {
            this.showNotification('Transação sem identificador — recarregue antes de salvar.', 'error');
            return;
        }

        try {
            const res = await this.authorizedFetch(`${API_URL}/dif/non-recurring/${route}`, {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ idParcela: item.idParcela, [field]: value })
            });

            if (res.status === 404) {
                this.showNotification('Essa transação não está mais na homologação — recarregue.', 'error');
                return;
            }
            if (!res.ok) {
                const txt = await res.text();
                throw new Error(txt || `Falha ao salvar ${label}`);
            }

            this.state.nonRecurringDif = this.state.nonRecurringDif.map(it =>
                it.difRowIndex === difRowIndex ? { ...it, [field]: value } : it
            );
            this.renderImport();
            this.showNotification(`${label[0].toUpperCase()}${label.slice(1)} atualizada.`, 'success');
        } catch (err) {
            console.error(err);
            this.showNotification(`Erro ao salvar ${label}: ${err.message}`, 'error');
        }
    },

    saveNonRecurringCategory(difRowIndex) {
        return this.saveNonRecurringField(difRowIndex, {
            route: 'category', field: 'categoria', label: 'categoria', inputId: 'cat',
        });
    },

    saveNonRecurringDate(difRowIndex) {
        return this.saveNonRecurringField(difRowIndex, {
            route: 'date', field: 'data', label: 'data', inputId: 'date',
        });
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
            this.showNotification('Transação copiada para ES.', 'success');
            await this.loadQueue();
        } catch (err) {
            console.error(err);
            this.showNotification(`Erro ao copiar para ES: ${err.message}`, 'error');
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
            this.showNotification('Transação movida para REJ.', 'success');
            await this.loadQueue();
        } catch (err) {
            console.error(err);
            this.showNotification(`Erro ao mover para REJ: ${err.message}`, 'error');
        }
    },

    async copyAllNonRecurringToES() {
        const total = (this.state.nonRecurringDif || []).length;
        if (total === 0) return;
        if (!confirm(`Copiar ${total} transação(ões) não recorrente(s) para ES?`)) return;

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
            this.showNotification(`${moved} transação(ões) copiada(s) para ES.`, 'success');
            await this.loadQueue();
        } catch (err) {
            console.error(err);
            this.showNotification(`Erro ao copiar todas para ES: ${err.message}`, 'error');
        }
    },

    // ── Opções de execução (barra de importação / popover) ──
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
        const date = options.startDate ? new Date(`${options.startDate}T00:00:00`).toLocaleDateString('pt-BR') : '-';
        const excludedCount = (options.excludeCategories || []).length;

        const summaryEl = document.getElementById('execution-options-summary');
        if (summaryEl) summaryEl.textContent = `Data inicial: ${date} | ${excludedCount} categoria(s) excluída(s)`;

        // Reflete a config nos chips compactos da barra de importação.
        const chipDate = document.getElementById('chip-date');
        if (chipDate) chipDate.textContent = date;
        const chipExcl = document.getElementById('chip-excl');
        if (chipExcl) chipExcl.textContent = excludedCount === 0
            ? 'nenhuma excluída'
            : `${excludedCount} categ. excluída${excludedCount > 1 ? 's' : ''}`;
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
