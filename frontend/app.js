import { DEFAULT_EXECUTION_OPTIONS } from './constants.js';
import { apiModule } from './api.js';
import { uiModule } from './ui.js';
import { authModule } from './auth.js';
import { queueModule } from './queue.js';
import { detailsModule } from './details.js';
import { executionModule } from './execution.js';

const app = Object.assign(
    {
        state: {
            authenticated: false,
            currentView: 'queue',
            conciliations: [],
            nonRecurringDif: [],
            currentExecution: null,
            executionHistory: [],
            statusPollingInterval: null,
            executionOptions: { ...DEFAULT_EXECUTION_OPTIONS },
            pendingCategoryEdits: {},
            pendingDateEdits: {},
            notificationTimeoutId: null,
            notificationHideTimeoutId: null,
            activeTab: 'conc',
            concSelected: new Set(),
            impSelected: new Set(),
            importPopoverOpen: false
        },

        async init() {
            await this.checkSession();
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

            // 'queue' renderiza o layout de abas (Conciliação + Importação).
            // O nome interno da view segue 'queue' para não tocar em auth.js /
            // execution.js, que reagem a `currentView === 'queue'`.
            if (view === 'queue') {
                const template = document.getElementById('view-tabs').content.cloneNode(true);
                main.innerHTML = '';
                main.appendChild(template);
                this.state.concSelected = new Set();
                this.state.impSelected = new Set();
                this.state.activeTab = 'conc';
                this.initializeExecutionOptionsUI();
                this.switchTab('conc');
                this.loadQueue();
            }
        },
    },
    apiModule,
    uiModule,
    authModule,
    queueModule,
    detailsModule,
    executionModule,
);

// Expose on window so inline HTML event handlers (app.login(), app.switchTab()) work.
window.app = app;

// Fecha o popover de importação ao clicar fora da barra de importação.
document.addEventListener('click', (e) => {
    if (!e.target.closest('.import-bar')) app.closeImportPopover();
});

app.init();
