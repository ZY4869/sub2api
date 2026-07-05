export function mapErrorCategory(phase?: string | null, errType?: string | null): string {
  switch (String(phase || "").toLowerCase()) {
    case "auth":
      return "auth";
    case "routing":
      return "service_unavailable";
    case "upstream":
    case "network":
      return "upstream";
    case "internal":
      return "internal";
    case "request":
      switch (String(errType || "").toLowerCase()) {
        case "rate_limit_error":
          return "rate_limit";
        case "billing_error":
        case "subscription_error":
          return "quota";
        case "invalid_request_error":
          return "invalid_request";
        case "cyber_policy":
          return "cyber";
      }
  }
  return "other";
}
