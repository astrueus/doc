package middleware

// Register installs HTTP filters once, before router.Init() / web.Run().
// Order: session cookie check → response headers → auth → (rate limit placeholder).
func Register() {
	registerSession()
	registerHeaders()
	registerAuth()
	registerRateLimit()
}
