export default {
  billingProbe: {
    title: 'Upstream Billing Probe',
    description: 'Control concurrency and timeout for manual admin upstream billing probes. Account lists and usage reads do not trigger probes.',
    enabled: 'Enable billing probes',
    enabledHint: 'When disabled, admins cannot run single-account or batch upstream billing probes.',
    batchConcurrency: 'Batch concurrency',
    batchConcurrencyHint: 'Accounts processed at the same time during batch probes. Maximum 10.',
    timeoutSeconds: 'Per-account timeout',
    timeoutSecondsHint: 'Maximum wait time for each upstream billing probe. Maximum 120 seconds.',
    saved: 'Billing probe settings saved',
    loadFailed: 'Failed to load billing probe settings',
    saveFailed: 'Failed to save billing probe settings',
  },
}
