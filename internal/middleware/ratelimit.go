package middleware

// registerRateLimit is reserved for Round 3 MCP HTTP rate limiting.
// No-op in Round 2 so callers can keep a stable Register() order.
func registerRateLimit() {}
