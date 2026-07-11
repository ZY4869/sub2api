export default {
  title: '批量生图',
  hint: '全局开关启用后，允许此 Gemini 分组处理用户提交的批量生图任务。',
  allowedProviders: '允许的 provider',
  allowedProvidersHint: '用逗号分隔；留空表示不额外限制 provider 策略。',
  allowedModels: '允许的模型',
  allowedModelsPlaceholder: 'gemini-2.5-flash-image',
  allowedModelsHint: '每行一个展示模型 ID；留空表示仅使用分组模型策略。',
  maxItems: '最大条目数',
  maxDownloadMB: '最大 ZIP MB',
  downloadConcurrency: '下载并发',
}
