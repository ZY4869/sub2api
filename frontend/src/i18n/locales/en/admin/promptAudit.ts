export default {
  title: 'Prompt Audit',
  description: 'Configure Qwen3Guard prompt safety auditing and review blocked, flagged, and async audit events.',
  runtime: {
    mode: 'Mode',
    queue: 'Queue',
    queueDetail: '{processing} processing, {failed} failed',
    workers: 'Workers',
    capacity: 'Capacity {capacity}',
    metrics: 'Blocks',
    metricsDetail: '{allow} allowed, {flag} flagged'
  },
  config: {
    title: 'Audit Configuration',
    description: 'Disabled by default. Probe Guard endpoints before enabling blocking mode.',
    enabled: 'Enable prompt audit',
    blocking: 'Block high-risk requests synchronously',
    storePass: 'Store pass events',
    strategy: 'Strategy',
    workerCount: 'Worker count',
    queueCapacity: 'Queue capacity',
    scanners: 'Scanner categories',
    selectAllScanners: 'Select all',
    allGroups: 'Apply to all groups',
    groupIds: 'Group IDs',
    endpoints: 'Guard endpoint pool',
    addEndpoint: 'Add endpoint',
    noEndpoints: 'No endpoints configured. Saving is allowed while disabled; enabling requires at least one endpoint.'
  },
  endpoint: {
    id: 'Endpoint ID',
    name: 'Name',
    baseUrl: 'Base URL',
    model: 'Model',
    timeout: 'Timeout ms',
    inputLimit: 'Input limit',
    enabled: 'Enable endpoint',
    tokenMode: 'Token',
    keepToken: 'Keep',
    replaceToken: 'Replace',
    clearToken: 'Clear',
    token: 'New token',
    tokenStored: 'Token stored',
    tokenMissing: 'No stored token',
    tokenWillReplace: 'Token will be replaced on save',
    tokenWillClear: 'Token will be cleared on save',
    probe: 'Probe',
    probing: 'Probing...'
  },
  scanners: {
    violent: 'Violent',
    non_violent_illegal_acts: 'Non-violent illegal acts',
    sexual_content_or_sexual_acts: 'Sexual content',
    pii: 'PII',
    suicide_and_self_harm: 'Suicide and self-harm',
    unethical_acts: 'Unethical acts',
    politically_sensitive_topics: 'Politically sensitive',
    copyright_violation: 'Copyright violation',
    jailbreak: 'Jailbreak'
  },
  filters: {
    requestId: 'Request ID',
    promptHash: 'Prompt hash',
    keyword: 'Keyword',
    allDecisions: 'All decisions',
    allRisks: 'All risks'
  },
  decision: {
    pass: 'Pass',
    flag: 'Flag',
    critical: 'Critical'
  },
  risk: {
    low: 'Low',
    medium: 'Medium',
    high: 'High',
    critical: 'Critical'
  },
  events: {
    empty: 'No prompt audit events yet',
    previewDelete: 'Preview delete',
    deletePreview: '{count} events match the current filter.',
    confirmDeleteFilter: 'Confirm delete',
    columns: {
      createdAt: 'Time',
      decision: 'Decision',
      target: 'Target',
      preview: 'Redacted preview',
      request: 'Request',
      actions: 'Actions'
    }
  },
  detail: {
    title: 'Event #{id}',
    decision: 'Decision / Risk',
    target: 'Provider / Model',
    user: 'User',
    group: 'Group',
    endpoint: 'Endpoint',
    guard: 'Guard endpoint',
    promptHash: 'Prompt hash',
    categories: 'Categories',
    preview: 'Redacted preview',
    fullPrompt: 'Full prompt'
  },
  messages: {
    loadFailed: 'Failed to load prompt audit',
    saveFailed: 'Failed to save prompt audit config',
    saved: 'Prompt audit config saved',
    probeOk: 'Guard endpoint is reachable',
    probeFailed: 'Guard endpoint probe failed',
    detailFailed: 'Failed to load event details',
    deleteFailed: 'Failed to delete audit events',
    deleted: 'Audit events deleted',
    previewFailed: 'Failed to preview delete',
    stepUpConfigPrompt: 'This action changes prompt audit policy. Enter the current admin 2FA code.',
    stepUpDeletePrompt: 'This action deletes prompt audit events. Enter the current admin 2FA code.',
    stepUpRequired: 'A 2FA code is required to continue.'
  }
}
