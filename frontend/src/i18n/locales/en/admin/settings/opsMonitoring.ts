export default {
    opsMonitoring: {
        title: "Ops Monitoring",
        description: "Enable ops monitoring for troubleshooting and health visibility",
        disabled: "Ops monitoring is disabled",
        enabled: "Enable Ops Monitoring",
        enabledHint: "Enable the ops monitoring module (admin only)",
        realtimeEnabled: "Enable Realtime Monitoring",
        realtimeEnabledHint: "Enable realtime QPS/metrics push (WebSocket)",
        queryMode: "Default Query Mode",
        queryModeHint: "Default query mode for Ops Dashboard (auto/raw/preagg)",
        queryModeAuto: "Auto (recommended)",
        queryModeRaw: "Raw (most accurate, slower)",
        queryModePreagg: "Preagg (fastest, requires aggregation)",
        metricsInterval: "Metrics Collection Interval (seconds)",
        metricsIntervalHint: "How often to collect system/request metrics (60-3600 seconds)",
    }
}
