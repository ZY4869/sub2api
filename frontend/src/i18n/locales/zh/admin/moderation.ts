export default {
  title: '内容审核审计',
  description: '查看文本请求入口的审核审计记录、去重复用情况和 fail-open 错误。',
  filters: {
    requestId: '请求 ID',
    clientRequestId: '客户端请求 ID',
    provider: 'Provider',
    model: '模型',
    sourceEndpoint: '来源入口',
    contentHash: '内容哈希',
    userId: '用户 ID',
    hit: '命中状态',
    allHits: '全部',
    hitOnly: '仅命中',
    passOnly: '仅未命中',
    searchPlaceholder: '按 request_id / client_request_id / model 筛选'
  },
  columns: {
    createdAt: '时间',
    provider: 'Provider / 模型',
    sourceEndpoint: '来源入口',
    summary: '脱敏摘要',
    request: '关联请求',
    status: '结果',
    latency: '耗时'
  },
  status: {
    hit: '命中',
    pass: '未命中',
    dedupe: '复用最近结论',
    error: '错误'
  },
  detail: {
    title: '审核详情',
    requestId: '请求 ID',
    clientRequestId: '客户端请求 ID',
    userId: '用户 ID',
    apiKeyId: 'API Key ID',
    contentHash: '内容哈希',
    summary: '脱敏摘要',
    latency: '耗时',
    errorReason: '错误原因',
    categories: '分类'
  },
  empty: '暂无审核记录',
  loadFailed: '加载审核记录失败',
  detailFailed: '加载审核详情失败'
}
