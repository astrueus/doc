package mcpdto

type GetDocumentIn struct {
	DocumentID   int    `json:"document_id,omitempty" jsonschema:"文档 ID（与 book_identify+doc_identify 二选一）"`
	BookIdentify string `json:"book_identify,omitempty" jsonschema:"项目 identify"`
	DocIdentify  string `json:"doc_identify,omitempty" jsonschema:"文档 identify"`
}

type GetDocumentOut struct {
	DocumentID   int    `json:"document_id"`
	BookID       int    `json:"book_id"`
	BookIdentify string `json:"book_identify"`
	Title        string `json:"title"`
	Identify     string `json:"identify"`
	Markdown     string `json:"markdown"`
	Release      string `json:"release"`
	Version      int64  `json:"version"`
	ParentID     int    `json:"parent_id"`
}

type CreateDocumentIn struct {
	BookID   int    `json:"book_id" jsonschema:"项目 ID"`
	ParentID int    `json:"parent_id,omitempty" jsonschema:"父文档 ID，0 表示根"`
	Title    string `json:"title" jsonschema:"文档标题"`
	Identify string `json:"identify,omitempty" jsonschema:"文档唯一标识，可空自动生成"`
	Markdown string `json:"markdown,omitempty" jsonschema:"初始 Markdown 内容"`
}

type CreateDocumentOut struct {
	DocumentID int   `json:"document_id"`
	Version    int64 `json:"version"`
}

type UpdateDocumentContentIn struct {
	DocumentID    int    `json:"document_id" jsonschema:"文档 ID"`
	ExpectVersion int64  `json:"expect_version" jsonschema:"期望的 Document.Version，乐观锁用"`
	Markdown      string `json:"markdown" jsonschema:"完整 Markdown 内容（覆盖式写入）"`
	AutoRelease   bool   `json:"auto_release,omitempty" jsonschema:"写入后是否立即 release（Markdown→HTML），默认 false"`
}

type UpdateDocumentContentOut struct {
	DocumentID int    `json:"document_id"`
	Version    int64  `json:"version"`
	Message    string `json:"message"`
}

type AppendDocumentContentIn struct {
	DocumentID     int    `json:"document_id" jsonschema:"文档 ID"`
	ExpectVersion  int64  `json:"expect_version" jsonschema:"期望的 Document.Version，乐观锁用"`
	MarkdownAppend string `json:"markdown_append" jsonschema:"追加的 Markdown 内容"`
	AutoRelease    bool   `json:"auto_release,omitempty" jsonschema:"写入后是否立即 release（Markdown→HTML），默认 false"`
}

type AppendDocumentContentOut struct {
	DocumentID int   `json:"document_id"`
	Version    int64 `json:"version"`
}

type UpdateDocumentMetaIn struct {
	DocumentID int     `json:"document_id" jsonschema:"文档 ID"`
	Title      *string `json:"title,omitempty" jsonschema:"新标题"`
	Identify   *string `json:"identify,omitempty" jsonschema:"新 identify"`
	OrderSort  *int    `json:"order_sort,omitempty" jsonschema:"排序值"`
	ParentID   *int    `json:"parent_id,omitempty" jsonschema:"父文档 ID"`
}

type UpdateDocumentMetaOut struct {
	DocumentID int `json:"document_id"`
}

type ReleaseDocumentIn struct {
	DocumentID int `json:"document_id,omitempty" jsonschema:"文档 ID（与 book_id 二选一）"`
	BookID     int `json:"book_id,omitempty" jsonschema:"项目 ID，批量 release 整个 book"`
}

type ReleaseDocumentOut struct {
	ReleasedCount int `json:"released_count"`
}

type DeleteDocumentIn struct {
	DocumentID int  `json:"document_id" jsonschema:"文档 ID"`
	Confirm    bool `json:"confirm" jsonschema:"必须为 true 才会真删"`
}

type DeleteDocumentOut struct {
	DeletedCount      int `json:"deleted_count"`
	SnapshotHistoryID int `json:"snapshot_history_id"`
}
