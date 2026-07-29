package template_fun

func Asset(p string, cdn string) string {
	return cdn + p
}
