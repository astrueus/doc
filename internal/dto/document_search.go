package dto

import "time"

// DocumentSearchResult 搜索展示结构（LIKE 查询结果，引擎仍冻结）。
type DocumentSearchResult struct {
	DocumentId   int       `json:"doc_id"`
	DocumentName string    `json:"doc_name"`
	Identify     string    `json:"identify"`
	Description  string    `json:"description"`
	Author       string    `json:"author"`
	ModifyTime   time.Time `json:"modify_time"`
	CreateTime   time.Time `json:"create_time"`
	BookId       int       `json:"book_id"`
	BookName     string    `json:"book_name"`
	BookIdentify string    `json:"book_identify"`
	SearchType   string    `json:"search_type"`
}

// NewDocumentSearchResult 返回空的搜索展示结构。
func NewDocumentSearchResult() *DocumentSearchResult {
	return &DocumentSearchResult{}
}
