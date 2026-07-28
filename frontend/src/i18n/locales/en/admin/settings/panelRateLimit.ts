export default {
  panelRateLimit: {
    title: 'Panel API Rate Limits',
    description: 'Control per-minute request limits for admin, user, and public panel APIs.',
    enabled: 'Enable panel rate limits',
    enabledHint: 'Disabled by default. Redis failures do not block panel requests.',
    userRpm: 'User API RPM',
    heavyRpm: 'Aggregate API RPM',
    publicIpRpm: 'Public IP RPM',
    exemptAdmin: 'Exempt admins',
    exemptAdminHint: 'Admin accounts bypass user and aggregate panel limits when enabled.',
    saved: 'Panel rate limit settings saved',
    loadFailed: 'Failed to load panel rate limit settings',
    saveFailed: 'Failed to save panel rate limit settings',
  },
}
