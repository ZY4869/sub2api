export default {
    title: "Claude Code 客户端限制",
    tooltip: "启用后，此分组仅允许 Claude Code 官方客户端访问。非 Claude Code 请求将被拒绝或降级到指定分组。",
    enabled: "仅限 Claude Code",
    disabled: "允许所有客户端",
    fallbackGroup: "降级分组",
    fallbackHint: "非 Claude Code 请求将使用此分组，留空则直接拒绝",
    noFallback: "不降级（直接拒绝）",
}
