export default {
  title: "OpenAI runtime policy",
  allowLive: "Allow Live sessions",
  allowLiveHint: "Disabled by default. Enables /v1/live and Codex realtime WebSocket for this group.",
  maxReasoningEffort: "Maximum reasoning effort",
  maxReasoningEffortHint: "Requests will not exceed this group limit. Leave empty for no group cap.",
  mappingTitle: "Reasoning effort mappings",
  mappingHint: "Match public model IDs, with * wildcard support. Matching requests are mapped to the target effort.",
  modelPattern: "Model ID",
  from: "From",
  to: "To",
  anyEffort: "Any",
  noLimit: "No limit",
  addMapping: "Add mapping",
  removeMapping: "Remove mapping"
}
