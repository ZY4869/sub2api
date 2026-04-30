export default {
    title: "Request Details",
    description: "Inspect gateway traces across inbound, normalized, upstream, and response payloads with searchable diagnostics.",
    pageTabs: {
        trace: "Request Details",
        subject: "Account Details",
    },
    actions: {
        exportMasked: "Export CSV",
        exportRaw: "Export Raw CSV",
        cleanupFilter: "Cleanup (Current Filter)",
        cleanupExpired: "Cleanup Expired Data",
    },
    cleanup: {
        confirmFilter: "Are you sure you want to cleanup request details matching the current filter? This action cannot be undone.",
        confirmExpired: "Are you sure you want to cleanup expired request details now? This action cannot be undone.",
        success: "Cleanup complete: deleted {traces} traces / {audits} audits",
        successExpired: "Cleanup complete: deleted {traces} traces / {audits} audits (cutoff={cutoff})",
        failed: "Failed to cleanup request details",
    },
    summary: {
        requests: "Requests",
        successErrorHint: "Success {success} / Error {error}",
        latency: "Latency",
        capability: "Capability Coverage",
        capabilityHint: "Stream {stream} / Tools {tools} / Thinking {thinking}",
        rawCoverage: "Raw Coverage",
        rawCoverageHint: "{raw} of {total} traces include raw payloads",
    }
}
