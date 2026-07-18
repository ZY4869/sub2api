export default {
  title: '提示词审计',
  description: '配置 Qwen3Guard 提示词安全审计，复核阻止、标记和异步审计事件。',
  runtime: {
    mode: '模式',
    queue: '队列',
    queueDetail: '处理中 {processing}，失败 {failed}',
    workers: 'Worker',
    capacity: '容量 {capacity}',
    metrics: '阻止数',
    metricsDetail: '放行 {allow}，标记 {flag}'
  },
  config: {
    title: '审计配置',
    description: '默认关闭；启用阻止模式前请先探测 Guard 端点。',
    enabled: '启用提示词审计',
    blocking: '同步阻止高危请求',
    storePass: '保存通过事件',
    strategy: '策略',
    workerCount: 'Worker 数',
    queueCapacity: '队列容量',
    scanners: '扫描类别',
    selectAllScanners: '选择全部',
    allGroups: '应用到全部分组',
    groupIds: '分组 ID',
    endpoints: 'Guard 端点池',
    addEndpoint: '添加端点',
    noEndpoints: '未配置端点。关闭状态可以保存；启用前至少需要一个可用端点。'
  },
  endpoint: {
    id: '端点 ID',
    name: '名称',
    baseUrl: 'Base URL',
    model: '模型',
    timeout: '超时 ms',
    inputLimit: '输入上限',
    enabled: '启用端点',
    tokenMode: 'Token',
    keepToken: '保留',
    replaceToken: '替换',
    clearToken: '清除',
    token: '新 Token',
    tokenStored: '已保存 token',
    tokenMissing: '未保存 token',
    tokenWillReplace: '保存时替换 token',
    tokenWillClear: '保存时清除 token',
    probe: '探测',
    probing: '探测中...'
  },
  scanners: {
    violent: '暴力',
    non_violent_illegal_acts: '非暴力违法',
    sexual_content_or_sexual_acts: '性内容',
    pii: '个人敏感信息',
    suicide_and_self_harm: '自杀与自残',
    unethical_acts: '不道德行为',
    politically_sensitive_topics: '政治敏感',
    copyright_violation: '版权侵权',
    jailbreak: '越狱攻击'
  },
  filters: {
    requestId: '请求 ID',
    promptHash: '提示词哈希',
    keyword: '关键词',
    allDecisions: '全部结果',
    allRisks: '全部风险'
  },
  decision: {
    pass: '通过',
    flag: '标记',
    critical: '严重'
  },
  risk: {
    low: '低',
    medium: '中',
    high: '高',
    critical: '严重'
  },
  events: {
    empty: '暂无提示词审计事件',
    previewDelete: '删除预览',
    deletePreview: '当前筛选条件匹配 {count} 条事件。',
    confirmDeleteFilter: '确认删除',
    columns: {
      createdAt: '时间',
      decision: '结果',
      target: '目标',
      preview: '脱敏预览',
      request: '请求',
      actions: '操作'
    }
  },
  detail: {
    title: '事件 #{id}',
    decision: '结果 / 风险',
    target: 'Provider / 模型',
    user: '用户',
    group: '分组',
    endpoint: '入口',
    guard: 'Guard 端点',
    promptHash: '提示词哈希',
    categories: '分类',
    preview: '脱敏预览',
    fullPrompt: '完整提示词'
  },
  messages: {
    loadFailed: '加载提示词审计失败',
    saveFailed: '保存提示词审计配置失败',
    saved: '提示词审计配置已保存',
    probeOk: 'Guard 端点可用',
    probeFailed: 'Guard 端点探测失败',
    detailFailed: '加载事件详情失败',
    deleteFailed: '删除审计事件失败',
    deleted: '审计事件已删除',
    previewFailed: '删除预览失败',
    stepUpConfigPrompt: '此操作会修改提示词审计策略，请输入当前管理员的 2FA 验证码。',
    stepUpDeletePrompt: '此操作会删除提示词审计事件，请输入当前管理员的 2FA 验证码。',
    stepUpRequired: '需要 2FA 验证码才能继续。'
  }
}
