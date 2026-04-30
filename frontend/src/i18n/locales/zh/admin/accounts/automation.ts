export default {
    autoRecoveryProbe: {
        headline: "恢复探测：{status}",
        checkedAt: "最近探测：{time}",
        nextRetryAt: "下次重试：{time}",
        errorCode: "错误代码：{code}",
        autoBlacklisted: "已自动拉黑",
        successIndicator: "该账号已通过恢复探测并恢复正常",
        statuses: {
            success: "已恢复",
            retry_scheduled: "稍后重试",
            blacklisted: "已拉黑",
            unknown: "已记录",
        },
        summaries: {
            success: "7 天限流恢复窗口到期后，后台已自动探测并恢复账号。",
            retry_scheduled: "本次恢复探测遇到临时错误，系统会在 30 分钟后自动再试。",
            blacklisted: "本次恢复探测确认账号仍不可用，系统已自动拉黑。",
            unknown: "最近一次 7 天限流恢复探测结果已记录。",
        },
    }
}
