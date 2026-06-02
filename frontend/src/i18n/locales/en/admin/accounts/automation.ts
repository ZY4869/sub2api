export default {
  autoRecoveryProbe: {
    headline: "Recovery probe: {status}",
    checkedAt: "Last probe: {time}",
    nextRetryAt: "Next retry: {time}",
    errorCode: "Error code: {code}",
    autoBlacklisted: "Auto blacklisted",
    successIndicator:
      "This account passed the recovery probe and is healthy again",
    statuses: {
      success: "Recovered",
      retry_scheduled: "Retry scheduled",
      blacklisted: "Blacklisted",
      unknown: "Recorded",
    },
    summaries: {
      success:
        "The backend retried the account automatically after the 7-day limit window expired and restored it successfully.",
      retry_scheduled:
        "The recovery probe hit a temporary error and will retry automatically in 30 minutes.",
      blacklisted:
        "The recovery probe confirmed the account is still unusable and blacklisted it automatically.",
      unknown:
        "The latest 7-day limit recovery probe result has been recorded.",
    },
  },
  daily5h: {
    toolbarLabel: "Daily 5H",
    toolbarHint: "Trigger the 5H window automatically at 07:00 every day",
    settingsButtonTitle: "Configure daily 5H trigger",
    dialogTitle: "Daily 07:00 5H Trigger",
    dialogSummaryTitle: "How automatic triggering works",
    dialogSummaryBody:
      "At 07:00 every morning in the server local timezone, the system runs one text test for eligible accounts in the selected account types that are not currently rate limited. The request text is fixed to Output exactly: OK.",
    enableLabel: "Enable daily automatic trigger",
    enableHint:
      "When enabled, it runs at most once per day. If the service restarts before the run, it can catch up later that same day.",
    includePausedLabel: "Include paused accounts",
    includePausedHint:
      "When enabled, disabled or unschedulable accounts that are not currently rate limited are included too. Blacklisted and currently limited accounts are still skipped.",
    ignoreFreeLabel: "Ignore Free accounts",
    ignoreFreeHint:
      "When enabled, ChatGPT Free accounts do not participate in the daily 5H trigger. Recommended for accounts whose quotas refresh weekly or monthly.",
    accountTypesLabel: "Account types to trigger",
    accountTypesHint:
      "Only the selected account types will participate in the daily 5H trigger. ChatGPT OAuth is enabled by default.",
    accountTypeOpenAI: "ChatGPT OAuth",
    accountTypeOpenAIHint:
      "Automatically picks the latest visible Mini-family model for each account by default.",
    accountTypeAnthropic: "Claude Code OAuth / Setup Token",
    accountTypeAnthropicHint:
      "Automatically picks the latest visible Haiku-family model for each account by default.",
    accountTypeGemini: "OAuth (Google)",
    accountTypeGeminiHint:
      "Automatically picks the latest visible Gemini-family model for each account by default.",
    candidateCount: "{count} candidate accounts",
    modelCount: "{count} candidate models",
    modelConfigLabel: "Model selection strategy",
    modelConfigHint:
      "Auto mode picks the latest model from the target family within each account's own whitelist. Fixed mode only uses the specific family model you choose.",
    familyHintOpenAI: "Only Mini-family models are shown",
    familyHintAnthropic: "Only Haiku-family models are shown",
    familyHintGemini: "Only Gemini-family models are shown",
    modelCandidatesHint: "{count} fixed-model candidates are currently available",
    modelModeAuto: "Auto latest",
    modelModeAutoHint:
      "Choose the latest allowed model from the target family for each account.",
    modelModeFixed: "Fixed model",
    modelModeFixedHint:
      "Always request your selected family model. If an account cannot see it, that account is skipped for the day.",
    fixedModelPlaceholder: "Select a fixed model",
    supportedAccountsCount: "{count} supporting accounts",
    noFamilyModelsHint:
      "No eligible fixed model is currently available in this family. Adjust account whitelists or switch back to auto mode.",
    loadFailed: "Failed to load daily 5H settings",
    updateSuccess: "Daily 5H trigger settings updated",
    updateFailed: "Failed to update daily 5H settings",
  },
};
