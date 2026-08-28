package dto

import "time"

// AttachmentResult 附件展示结构（字段平铺，不嵌入 model）。
type AttachmentResult struct {
	AttachmentId  int       `json:"attachment_id"`
	BookId        int       `json:"book_id"`
	DocumentId    int       `json:"doc_id"`
	FileName      string    `json:"file_name"`
	FilePath      string    `json:"file_path"`
	FileSize      float64   `json:"file_size"`
	HttpPath      string    `json:"http_path"`
	FileExt       string    `json:"file_ext"`
	CreateTime    time.Time `json:"create_time"`
	CreateAt      int       `json:"create_at"`
	IsExist       bool
	BookName      string
	DocumentName  string
	FileShortSize string
	Account       string
	LocalHttpPath string
}

// NewAttachmentResult 返回空的附件展示结构。
func NewAttachmentResult() *AttachmentResult {
	return &AttachmentResult{IsExist: false}
}
