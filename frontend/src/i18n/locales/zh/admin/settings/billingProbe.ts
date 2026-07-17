export default {
  billingProbe: {
    title: '上游计费探测',
    description: '控制管理员手动探测上游计费快照的并发与超时；账号列表和用量读取不会自动触发探测。',
    enabled: '启用计费探测',
    enabledHint: '关闭后，管理员无法发起单账号或批量上游计费探测。',
    batchConcurrency: '批量并发数',
    batchConcurrencyHint: '批量探测时同时处理的账号数量，最大 10。',
    timeoutSeconds: '单账号超时秒数',
    timeoutSecondsHint: '每个账号上游计费探测的最长等待时间，最大 120 秒。',
    saved: '计费探测设置保存成功',
    loadFailed: '加载计费探测设置失败',
    saveFailed: '保存计费探测设置失败',
  },
}
