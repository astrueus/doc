package router

// Init registers all HTTP routes. Call once before web.Run().
// Order: static/page routes first, then domain groups; keep param routes after
// more specific prefixes within each register* func.
func Init() {
	registerPage()
	registerAccount()
	registerManager()
	registerBook()
	registerBlog()
	registerDocument()
	registerAPI()
}
