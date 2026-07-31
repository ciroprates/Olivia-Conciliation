import { API_URL } from './constants.js';

// Ações por linha da aba Conciliação (DIF recorrente ↔ candidata ES).
// No layout de abas, os detalhes da transação passaram a ser exibidos inline na
// própria tabela (a candidata ES fica lado a lado com a DIF), então não há mais
// uma view separada de detalhes: aceitar/rejeitar acontecem direto na linha.
export const detailsModule = {
    // Aceitar: vincula a DIF à candidata ES escolhida. O backend ainda recebe um
    // array (`esRowIndices`) — ver comentário de domínio na issue #6 sobre a
    // futura mudança para índice único; aqui passamos exatamente uma candidata.
    async acceptConciliation(difRowIndex, esRowIndex) {
        try {
            const res = await this.authorizedFetch(`${API_URL}/conciliations/${difRowIndex}/accept`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ esRowIndices: [esRowIndex] })
            });

            if (!res.ok) {
                const txt = await res.text();
                throw new Error(txt || 'Falha ao conciliar');
            }

            this.showNotification('Conciliação realizada com sucesso!', 'success');
            await this.loadQueue();
        } catch (err) {
            console.error(err);
            this.showNotification(`Erro ao conciliar: ${err.message}`, 'error');
        }
    },

    async rejectConciliation(difRowIndex) {
        if (!confirm('Rejeitar esta parcela? A referência será movida para REJ.')) return;

        try {
            const res = await this.authorizedFetch(`${API_URL}/conciliations/${difRowIndex}/reject`, {
                method: 'POST'
            });

            if (!res.ok) {
                const txt = await res.text();
                throw new Error(txt || 'Falha ao rejeitar');
            }

            this.showNotification('Parcela rejeitada.', 'success');
            await this.loadQueue();
        } catch (err) {
            console.error(err);
            this.showNotification(`Erro ao rejeitar: ${err.message}`, 'error');
        }
    },
};
