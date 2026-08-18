package mcpdto

type ListBooksIn struct {
	Page     int `json:"page,omitempty" jsonschema:"页码，默认 1"`
	PageSize int `json:"page_size,omitempty" jsonschema:"每页条数，默认 20，最大 100"`
}

type BookBrief struct {
	BookID      int    `json:"book_id"`
	Identify    string `json:"identify"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	RoleID      int    `json:"role_id"`
	DocCount    int    `json:"doc_count"`
}

type ListBooksOut struct {
	Total int         `json:"total"`
	Items []BookBrief `json:"items"`
}

type ListDocumentTreeIn struct {
	BookID       int    `json:"book_id,omitempty" jsonschema:"项目 ID（与 book_identify 二选一）"`
	BookIdentify string `json:"book_identify,omitempty" jsonschema:"项目 identify"`
}

type DocumentTreeNode struct {
	DocumentID int    `json:"document_id"`
	ParentID   int    `json:"parent_id"`
	Title      string `json:"title"`
	Identify   string `json:"identify"`
	Version    int64  `json:"version"`
	OrderSort  int    `json:"order_sort"`
}

type ListDocumentTreeOut struct {
	BookID       int                `json:"book_id"`
	BookIdentify string             `json:"book_identify"`
	Nodes        []DocumentTreeNode `json:"nodes"`
}

type CreateBookIn struct {
	Title        string `json:"title" jsonschema:"项目名称"`
	Identify     string `json:"identify" jsonschema:"项目唯一标识"`
	Private      bool   `json:"private,omitempty" jsonschema:"true 表示私有，默认公开"`
	Description  string `json:"description,omitempty" jsonschema:"简介"`
	ItemIdentify string `json:"item_identify,omitempty" jsonschema:"所属空间 key，缺省为默认空间"`
}

type UpdateBookIn struct {
	BookID       int     `json:"book_id,omitempty" jsonschema:"项目 ID（与 book_identify 二选一）"`
	BookIdentify string  `json:"book_identify,omitempty" jsonschema:"项目 identify"`
	Title        *string `json:"title,omitempty" jsonschema:"新标题"`
	Description  *string `json:"description,omitempty" jsonschema:"新简介"`
	Private      *bool   `json:"private,omitempty" jsonschema:"是否私有；省略则不改"`
}
