export default {
  title: '审计日志',
  description: '查看管理员敏感操作、二次验证拦截和安全事件记录。',
  empty: '暂无审计日志',
  loadFailed: '加载审计日志失败',
  cleanupExpired: '清理过期日志',
  cleanupConfirm: '确定要清理超过保留期的审计日志吗？',
  cleanupSuccess: '已清理 {count} 条过期审计日志',
  cleanupFailed: '清理过期审计日志失败',
  stepUpTotpPrompt: '此操作会删除过期审计日志，请输入当前管理员的 2FA 验证码。',
  stepUpTotpRequired: '需要 2FA 验证码才能清理审计日志。',
  filters: {
    action: '操作',
    targetType: '目标类型',
    targetId: '目标 ID',
    requestId: '请求 ID',
    actorUserId: '操作者用户 ID',
    allStatuses: '全部状态'
  },
  columns: {
    createdAt: '时间',
    actor: '操作者',
    action: '操作',
    target: '目标',
    status: '状态',
    request: '请求',
    metadata: '元数据'
  },
  status: {
    success: '成功',
    denied: '已拒绝',
    failure: '失败'
  }
}
