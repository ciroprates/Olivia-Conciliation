const API_URL = 'http://localhost:8080/api';

const app = {
    state: {
        currentView: 'queue',
        conciliations: [],
        details: null,
        selectedCandidates: new Set()
    },

    init() {
        this.navigate('queue');
    },

    navigate(view) {
        this.state.currentView = view;
        const main = document.getElementById('main-content');

        if (view === 'queue') {
            const template = document.getElementById('view-queue').content.cloneNode(true);
            main.innerHTML = '';
            main.appendChild(template);
            this.loadQueue();

            // Search listener
            document.getElementById('search').addEventListener('input', (e) => {
                this.renderQueue(e.target.value);
            });
        } else if (view === 'details') {
            const template = document.getElementById('view-details').content.cloneNode(true);
            main.innerHTML = '';
            main.appendChild(template);
            this.renderDetails();
        }
    },

    async loadQueue() {
        try {
            const res = await fetch(`${API_URL}/conciliations`);
            const data = await res.json();
            this.state.conciliations = data || [];
            this.renderQueue();
        } catch (err) {
            console.error(err);
            alert('Erro ao carregar conciliações');
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
            const res = await fetch(`${API_URL}/conciliations/${id}`);
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
            const res = await fetch(`${API_URL}/conciliations/${difIndex}/accept`, {
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
            const res = await fetch(`${API_URL}/conciliations/${difIndex}/reject`, {
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
    }
};

// Start
app.init();
