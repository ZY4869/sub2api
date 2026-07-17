export default {
  title: 'Audit Logs',
  description: 'Review sensitive admin operations, step-up denials, and security events.',
  empty: 'No audit logs found',
  loadFailed: 'Failed to load audit logs',
  cleanupExpired: 'Clean Expired Logs',
  cleanupConfirm: 'Clean audit logs older than the retention window?',
  cleanupSuccess: 'Cleaned {count} expired audit log(s)',
  cleanupFailed: 'Failed to clean expired audit logs',
  stepUpTotpPrompt: 'This action deletes expired audit logs. Enter the current admin 2FA code.',
  stepUpTotpRequired: 'A 2FA code is required to clean audit logs.',
  filters: {
    action: 'Action',
    targetType: 'Target type',
    targetId: 'Target ID',
    requestId: 'Request ID',
    actorUserId: 'Actor user ID',
    allStatuses: 'All statuses'
  },
  columns: {
    createdAt: 'Time',
    actor: 'Actor',
    action: 'Action',
    target: 'Target',
    status: 'Status',
    request: 'Request',
    metadata: 'Metadata'
  },
  status: {
    success: 'Success',
    denied: 'Denied',
    failure: 'Failure'
  }
}
