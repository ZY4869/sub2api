export default {
    opsMonitoring: {
        title: "运维监控",
        description: "启用运维监控模块，用于排障与健康可视化",
        disabled: "运维监控已关闭",
        enabled: "启用运维监控",
        enabledHint: "启用运维监控模块（仅管理员可见）",
        realtimeEnabled: "启用实时监控",
        realtimeEnabledHint: "启用实时请求速率和指标推送（WebSocket）",
        queryMode: "默认查询模式",
        queryModeHint: "运维监控默认查询模式（自动/原始/预聚合）",
        queryModeAuto: "自动（推荐）",
        queryModeRaw: "原始（最准确，但较慢）",
        queryModePreagg: "预聚合（最快，需预聚合）",
        metricsInterval: "采集频率（秒）",
        metricsIntervalHint: "系统/请求指标采集频率（60-3600 秒）",
    },
    realtimeCountdown: {
        title: "实时倒计时",
        description: "控制当前账号在非账号页中的实时倒计时显示。",
        globalEnabled: "全站实时倒计时开关",
        globalEnabledHint: "关闭后仅冻结非账号页倒计时显示，账号页内的实时倒计时仍由账号页自己的开关控制。",
        scopeHint: "这里只影响 Ops 等非账号页的倒计时展示，不会影响账号页 More 菜单里的“账号页实时倒计时”开关。",
        saved: "全站实时倒计时偏好已保存",
        saveFailed: "保存全站实时倒计时偏好失败",
    }
}
