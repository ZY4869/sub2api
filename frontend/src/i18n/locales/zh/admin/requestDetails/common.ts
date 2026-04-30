export default {
    title: "请求详细",
    description: "按请求链路查看网关的入站、标准化、上游与最终响应详情，并支持筛选排障。",
    pageTabs: {
        trace: "请求详情",
        subject: "账号详细",
    },
    actions: {
        exportMasked: "导出脱敏 CSV",
        exportRaw: "导出原文 CSV",
        cleanupFilter: "按当前筛选清理",
        cleanupExpired: "立即清理过期数据",
    },
    cleanup: {
        confirmFilter: "确认按当前筛选条件清理请求详情？该操作不可撤销。",
        confirmExpired: "确认立即清理过期请求详情？该操作不可撤销。",
        success: "清理完成，删除 {traces} 条 trace / {audits} 条审计",
        successExpired: "清理完成，删除 {traces} 条 trace / {audits} 条审计（cutoff={cutoff}）",
        failed: "清理请求详情失败",
    },
    summary: {
        requests: "请求量",
        successErrorHint: "成功 {success} / 失败 {error}",
        latency: "延迟",
        capability: "能力覆盖率",
        capabilityHint: "流式 {stream} / 工具 {tools} / Thinking {thinking}",
        rawCoverage: "原文覆盖率",
        rawCoverageHint: "{raw} / {total} 条 trace 含原文快照",
    }
}
