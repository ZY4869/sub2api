export default {
  title: "OpenAI 运行策略",
  allowLive: "允许 Live 会话",
  allowLiveHint: "默认关闭。开启后此分组可处理 /v1/live 与 Codex realtime WebSocket。",
  maxReasoningEffort: "最大推理力度",
  maxReasoningEffortHint: "请求推理力度不会超过此上限；留空表示不设置分组上限。",
  mappingTitle: "推理力度映射",
  mappingHint: "按公开模型 ID 匹配，可用 * 通配符。匹配后把请求推理力度映射为目标力度。",
  modelPattern: "模型 ID",
  from: "来源力度",
  to: "目标力度",
  anyEffort: "任意",
  noLimit: "不限制",
  addMapping: "添加映射",
  removeMapping: "删除映射"
}
