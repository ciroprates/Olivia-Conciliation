export const API_URL = '/api';
export const EXECUTION_API_URL = '/executions';
export const EXECUTION_OPTIONS_STORAGE_KEY = 'olivia_execution_options_v1';

export const DEFAULT_EXECUTION_OPTIONS = {
    excludeCategories: ['Same person transfer', 'Credit card payment'],
    startDate: '2026-01-15'
};

export const DEFAULT_EXECUTION_PAYLOAD = {
    sheet: {
        enabled: true,
        tabName: 'Homologação'
    },
    artifacts: {
        csvEnabled: false
    }
};
