export default {
    autoRecoveryProbe: {
        headline: "Recovery probe: {status}",
        checkedAt: "Last probe: {time}",
        nextRetryAt: "Next retry: {time}",
        errorCode: "Error code: {code}",
        autoBlacklisted: "Auto blacklisted",
        successIndicator: "This account passed the recovery probe and is healthy again",
        statuses: {
            success: "Recovered",
            retry_scheduled: "Retry scheduled",
            blacklisted: "Blacklisted",
            unknown: "Recorded",
        },
        summaries: {
            success: "The backend retried the account automatically after the 7-day limit window expired and restored it successfully.",
            retry_scheduled: "The recovery probe hit a temporary error and will retry automatically in 30 minutes.",
            blacklisted: "The recovery probe confirmed the account is still unusable and blacklisted it automatically.",
            unknown: "The latest 7-day limit recovery probe result has been recorded.",
        },
    }
}
