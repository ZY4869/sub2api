export default {
    title: "Claude Code Client Restriction",
    tooltip: "When enabled, this group only allows official Claude Code clients. Non-Claude Code requests will be rejected or fallback to the specified group.",
    enabled: "Claude Code Only",
    disabled: "Allow All Clients",
    fallbackGroup: "Fallback Group",
    fallbackHint: "Non-Claude Code requests will use this group. Leave empty to reject directly.",
    noFallback: "No Fallback (Reject)",
}
