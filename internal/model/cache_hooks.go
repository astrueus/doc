package model

// AfterDocumentMutated 在文档插入 / 更新 / 删除成功后调用（bootstrap 注册缓存失效）。
var AfterDocumentMutated func(doc *Document)

// AfterBlogMutated 在博客保存 / 删除成功后调用。
var AfterBlogMutated func(blog *Blog)

func notifyDocumentMutated(doc *Document) {
	if doc == nil || AfterDocumentMutated == nil {
		return
	}
	AfterDocumentMutated(doc)
}

func notifyBlogMutated(blog *Blog) {
	if blog == nil || AfterBlogMutated == nil {
		return
	}
	AfterBlogMutated(blog)
}
