package errs

// Stable error codes for JSON APIs and MCP tools (Round 3).
const (
	CodeUnknown         = 6000
	CodeInternal        = 6001
	CodeInvalidParam    = 6002
	CodeUnauthorized    = 6003
	CodeForbidden       = 6004
	CodeNotFound        = 6005
	CodeVersionConflict = 6100 // Round 3 MCP optimistic lock
	CodeRateLimited     = 6200 // Round 3 MCP rate limit
	CodeConfirmRequired = 6300 // Round 3 MCP delete protection
)
