package mcpdto

type SearchDocumentIn struct {
	Query  string `json:"query" jsonschema:"全文搜索关键字"`
	BookID int    `json:"book_id,omitempty" jsonschema:"限定 book，0=全部可见"`
	Limit  int    `json:"limit,omitempty" jsonschema:"返回条数，默认 10，最大 50"`
}

type DocumentBrief struct {
	ID      int    `json:"id"`
	BookID  int    `json:"book_id"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	Version int64  `json:"version"`
}

type SearchDocumentOut struct {
	Total int             `json:"total"`
	Items []DocumentBrief `json:"items"`
}
