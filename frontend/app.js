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
            details: null,
            selectedCandidates: new Set(),
            currentExecution: null,
            executionHistory: [],
            statusPollingInterval: null,
            executionOptions: { ...DEFAULT_EXECUTION_OPTIONS },
            pendingCategoryEdits: {},
            pendingDateEdits: {},
            notificationTimeoutId: null,
            notificationHideTimeoutId: null
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

            if (view === 'queue') {
                const template = document.getElementById('view-queue').content.cloneNode(true);
                main.innerHTML = '';
                main.appendChild(template);
                this.initializeExecutionOptionsUI();
                this.loadQueue();

                const searchInput = document.getElementById('search');
                if (searchInput) {
                    searchInput.addEventListener('input', (e) => {
                        this.renderQueue(e.target.value);
                    });
                }

                const copyAllBtn = document.getElementById('btn-copy-all-non-recurring');
                if (copyAllBtn) {
                    copyAllBtn.addEventListener('click', () => this.copyAllNonRecurringToES());
                }
            } else if (view === 'details') {
                const template = document.getElementById('view-details').content.cloneNode(true);
                main.innerHTML = '';
                main.appendChild(template);
                document.getElementById('btn-back')?.addEventListener('click', () => this.navigate('queue'));
                document.getElementById('btn-reject')?.addEventListener('click', () => this.rejectCurrent());
                document.getElementById('btn-accept-selection')?.addEventListener('click', () => this.acceptSelection());
                this.renderDetails();
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

// Expose on window so inline HTML event handlers (app.login(), app.toggleCandidate()) work.
window.app = app;
app.init();
