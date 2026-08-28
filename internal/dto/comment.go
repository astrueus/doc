package dto

import "time"

// CommentResult 评论展示结构（字段平铺，不嵌入 model）。
type CommentResult struct {
	CommentId    int       `json:"comment_id"`
	Floor        int       `json:"floor"`
	BookId       int       `json:"book_id"`
	DocumentId   int       `json:"document_id"`
	Author       string    `json:"author"`
	MemberId     int       `json:"member_id"`
	IPAddress    string    `json:"ip_address"`
	CommentDate  time.Time `json:"comment_date"`
	Content      string    `json:"content"`
	Approved     int       `json:"approved"`
	UserAgent    string    `json:"user_agent"`
	ParentId     int       `json:"parent_id"`
	AgreeCount   int       `json:"agree_count"`
	AgainstCount int       `json:"against_count"`
	ReplyAccount string    `json:"reply_account"`
}
