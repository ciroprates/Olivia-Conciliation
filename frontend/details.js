import { API_URL } from './constants.js';

export const detailsModule = {
    async loadDetails(id) {
        try {
            const res = await this.authorizedFetch(`${API_URL}/conciliations/${id}`);
            const data = await res.json();
            this.state.details = data;
            this.state.selectedCandidates.clear();
            this.navigate('details');
        } catch (err) {
            console.error(err);
            this.showNotification('Erro ao carregar detalhes', 'error');
        }
    },

    renderDetails() {
        if (!this.state.details) return;

        const ref = this.state.details.reference;
        const candidates = this.state.details.candidates || [];

        const refContainer = document.getElementById('ref-details');
        refContainer.innerHTML = this.renderCardContent(ref);

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
                <input type="checkbox" ${isSelected ? 'checked' : ''}>
                <div class="candidate-details">
                    ${this.renderCardContent(c, true)}
                </div>
            `;
            el.querySelector('input[type="checkbox"]').addEventListener('change', () => this.toggleCandidate(c.rowIndex));
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
                <span class="label">Banco</span>
                <span>${item.banco || '-'}</span>
            </div>
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
            this.showNotification('Selecione pelo menos uma candidata.', 'error');
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
                this.showNotification('Conciliação realizada com sucesso!', 'success');
                this.navigate('queue');
            } else {
                const txt = await res.text();
                this.showNotification('Erro: ' + txt, 'error');
            }
        } catch (err) {
            console.error(err);
            this.showNotification('Erro na requisição', 'error');
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
                this.showNotification('Rejeitada com sucesso!', 'success');
                this.navigate('queue');
            } else {
                const txt = await res.text();
                this.showNotification('Erro: ' + txt, 'error');
            }
        } catch (err) {
            console.error(err);
            this.showNotification('Erro na requisição', 'error');
        }
    },
};
