export default {
  title: 'Content Moderation Audits',
  description: 'Review moderation audit records for text entry points, dedupe reuse, and fail-open outcomes.',
  filters: {
    requestId: 'Request ID',
    clientRequestId: 'Client request ID',
    provider: 'Provider',
    model: 'Model',
    sourceEndpoint: 'Source endpoint',
    contentHash: 'Content hash',
    userId: 'User ID',
    hit: 'Verdict',
    allHits: 'All',
    hitOnly: 'Hit only',
    passOnly: 'Pass only',
    searchPlaceholder: 'Filter by request_id / client_request_id / model'
  },
  columns: {
    createdAt: 'Time',
    provider: 'Provider / Model',
    sourceEndpoint: 'Source',
    summary: 'Redacted summary',
    request: 'Request link',
    status: 'Result',
    latency: 'Latency'
  },
  status: {
    hit: 'Hit',
    pass: 'Pass',
    dedupe: 'Reused recent verdict',
    error: 'Error'
  },
  detail: {
    title: 'Audit details',
    requestId: 'Request ID',
    clientRequestId: 'Client request ID',
    userId: 'User ID',
    apiKeyId: 'API Key ID',
    contentHash: 'Content hash',
    summary: 'Redacted summary',
    latency: 'Latency',
    errorReason: 'Error reason',
    categories: 'Categories'
  },
  empty: 'No audit records yet',
  loadFailed: 'Failed to load moderation audits',
  detailFailed: 'Failed to load moderation audit details'
}
