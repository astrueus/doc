package dto

import (
	"encoding/json"
	"time"

	"git.itopcms.com/astrueus/doc/internal/config"
)

// BookResult 项目展示结构（原 internal/model.BookResult）。
type BookResult struct {
	BookId         int       `json:"book_id"`
	BookName       string    `json:"book_name"`
	ItemId         int       `json:"item_id"`
	ItemName       string    `json:"item_name"`
	Identify       string    `json:"identify"`
	OrderIndex     int       `json:"order_index"`
	Description    string    `json:"description"`
	Publisher      string    `json:"publisher"`
	PrivatelyOwned int       `json:"privately_owned"`
	PrivateToken   string    `json:"private_token"`
	BookPassword   string    `json:"book_password"`
	DocCount       int       `json:"doc_count"`
	CommentStatus  string    `json:"comment_status"`
	CommentCount   int       `json:"comment_count"`
	CreateTime     time.Time `json:"create_time"`
	CreateName     string    `json:"create_name"`
	RealName       string    `json:"real_name"`
	ModifyTime     time.Time `json:"modify_time"`
	Cover          string    `json:"cover"`
	Theme          string    `json:"theme"`
	Label          string    `json:"label"`
	MemberId       int       `json:"member_id"`
	Editor         string    `json:"editor"`
	AutoRelease    bool      `json:"auto_release"`
	HistoryCount   int       `json:"history_count"`

	RoleId             config.BookRole `json:"role_id"`
	RoleName           string          `json:"role_name"`
	Status             int             `json:"status"`
	IsEnableShare      bool            `json:"is_enable_share"`
	IsUseFirstDocument bool            `json:"is_use_first_document"`

	LastModifyText   string `json:"last_modify_text"`
	IsDisplayComment bool   `json:"is_display_comment"`
	IsDownload       bool   `json:"is_download"`
	AutoSave         bool   `json:"auto_save"`
	Lang             string
}

// NewBookResult 返回空的项目展示结构。
func NewBookResult() *BookResult {
	return &BookResult{}
}

func (m *BookResult) String() string {
	ret, err := json.Marshal(*m)
	if err != nil {
		return ""
	}
	return string(ret)
}

// SetLang 设置 i18n 语言，供角色名翻译。
func (m *BookResult) SetLang(lang string) *BookResult {
	m.Lang = lang
	return m
}

// ConvertBookResult 导出文件路径。
type ConvertBookResult struct {
	PDFPath  string
	EpubPath string
	MobiPath string
	WordPath string
}
