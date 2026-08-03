package model

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"git.itopcms.com/jackliu/doc/internal/config"
	"git.itopcms.com/jackliu/doc/pkg/htmlutil"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"git.itopcms.com/jackliu/doc/internal/i18n"
)
var releaseQueue = make(chan int, 500)
var once = sync.Once{}

// Book struct .
type Book struct {
	BookId int `orm:"pk;auto;unique;column(book_id)" json:"book_id"`
	// BookName 项目名称.
	BookName string `orm:"column(book_name);size(500)" json:"book_name"`
	//所属项目空间
	ItemId int `orm:"column(item_id);type(int);default(1)" json:"item_id"`
	// Identify 项目唯一标识.
	Identify string `orm:"column(identify);size(100);unique" json:"identify"`
	//是否是自动发布 0 否/1 是
	AutoRelease int `orm:"column(auto_release);type(int);default(0)" json:"auto_release"`
	//是否开启下载功能 0 是/1 否
	IsDownload int `orm:"column(is_download);type(int);default(0)" json:"is_download"`
	OrderIndex int `orm:"column(order_index);type(int);default(0)" json:"order_index"`
	// Description 项目描述.
	Description string `orm:"column(description);size(2000)" json:"description"`
	//发行公司
	Publisher string `orm:"column(publisher);size(500)" json:"publisher"`
	Label     string `orm:"column(label);size(500)" json:"label"`
	// PrivatelyOwned 项目私有： 0 公开/ 1 私有
	PrivatelyOwned int `orm:"column(privately_owned);type(int);default(0)" json:"privately_owned"`
	// 当项目是私有时的访问Token.
	PrivateToken string `orm:"column(private_token);size(500);null" json:"private_token"`
	//访问密码.
	BookPassword string `orm:"column(book_password);size(500);null" json:"book_password"`
	//状态：0 正常/1 已删除
	Status int `orm:"column(status);type(int);default(0)" json:"status"`
	//默认的编辑器.
	Editor string `orm:"column(editor);size(50)" json:"editor"`
	// DocCount 包含文档数量.
	DocCount int `orm:"column(doc_count);type(int)" json:"doc_count"`
	// CommentStatus 评论设置的状态:open 为允许所有人评论，closed 为不允许评论, group_only 仅允许参与者评论 ,registered_only 仅允许注册者评论.
	CommentStatus string `orm:"column(comment_status);size(20);default(open)" json:"comment_status"`
	CommentCount  int    `orm:"column(comment_count);type(int)" json:"comment_count"`
	//封面地址
	Cover string `orm:"column(cover);size(1000)" json:"cover"`
	//主题风格
	Theme string `orm:"column(theme);size(255);default(default)" json:"theme"`
	// CreateTime 创建时间 .
	CreateTime time.Time `orm:"type(datetime);column(create_time);auto_now_add" json:"create_time"`
	//每个文档保存的历史记录数量，0 为不限制
	HistoryCount int `orm:"column(history_count);type(int);default(0)" json:"history_count"`
	//是否启用分享，0启用/1不启用
	IsEnableShare int       `orm:"column(is_enable_share);type(int);default(0)" json:"is_enable_share"`
	MemberId      int       `orm:"column(member_id);size(100)" json:"member_id"`
	ModifyTime    time.Time `orm:"type(datetime);column(modify_time);null;auto_now" json:"modify_time"`
	Version       int64     `orm:"type(bigint);column(version)" json:"version"`
	//是否使用第一篇文章项目为默认首页,0 否/1 是
	IsUseFirstDocument int `orm:"column(is_use_first_document);type(int);default(0)" json:"is_use_first_document"`
	//是否开启自动保存：0 否/1 是
	AutoSave int `orm:"column(auto_save);type(tinyint);default(0)" json:"auto_save"`
}

func (book *Book) String() string {
	ret, err := json.Marshal(*book)

	if err != nil {
		return ""
	}
	return string(ret)
}

// TableName 获取对应数据库表名.
func (book *Book) TableName() string {
	return "books"
}

// TableEngine 获取数据使用的引擎.
func (book *Book) TableEngine() string {
	return "INNODB"
}
func (book *Book) TableNameWithPrefix() string {
	return config.GetDatabasePrefix() + book.TableName()
}

func (book *Book) QueryTable() orm.QuerySeter {
	return orm.NewOrm().QueryTable(book.TableNameWithPrefix())
}

func NewBook() *Book {
	return &Book{}
}

// 添加一个项目
func (book *Book) Insert(lang string) error {
	o := orm.NewOrm()
	//	o.Begin()
	book.BookName = htmlutil.StripTags(book.BookName)
	if book.ItemId <= 0 {
		book.ItemId = 1
	}
	_, err := o.Insert(book)

	if err == nil {
		if book.Label != "" {
			NewLabel().InsertOrUpdateMulti(book.Label)
		}

		relationship := NewRelationship()
		relationship.BookId = book.BookId
		relationship.RoleId = 0
		relationship.MemberId = book.MemberId
		err = relationship.Insert()
		if err != nil {
			logs.Error("插入项目与用户关联 -> ", err)
			//o.Rollback()
			return err
		}
		document := NewDocument()
		document.BookId = book.BookId
		document.DocumentName = i18n.Tr(lang, "init.blank_doc") //"空白文档"
		document.MemberId = book.MemberId
		err = document.InsertOrUpdate()
		if err != nil {
			//o.Rollback()
			return err
		}
		//o.Commit()
		return nil
	}
	//o.Rollback()
	return err
}

func (book *Book) Find(id int, cols ...string) (*Book, error) {
	if id <= 0 {
		return book, ErrInvalidParameter
	}
	o := orm.NewOrm()

	err := o.QueryTable(book.TableNameWithPrefix()).Filter("book_id", id).One(book, cols...)

	return book, err
}

// 更新一个项目
func (book *Book) Update(cols ...string) error {
	o := orm.NewOrm()

	book.BookName = htmlutil.StripTags(book.BookName)
	temp := NewBook()
	temp.BookId = book.BookId

	if err := o.Read(temp); err != nil {
		return err
	}

	if book.Label != "" || temp.Label != "" {

		go NewLabel().InsertOrUpdateMulti(book.Label + "," + temp.Label)
	}

	_, err := o.Update(book, cols...)
	return err
}

func (book *Book) ThoroughDeleteBook(id int) error {
	if id <= 0 {
		return ErrInvalidParameter
	}
	o := orm.NewOrm()

	book, err := book.Find(id)
	if err != nil {
		return err
	}
	o.Begin()

	//删除附件,这里没有删除实际物理文件
	if err := o.DoTx(func(ctx context.Context, txOrm orm.TxOrmer) error {
		_, err = txOrm.Raw("DELETE FROM "+NewAttachment().TableNameWithPrefix()+" WHERE book_id=?", book.BookId).Exec()
		return err
	}); err != nil {
		return err
	}

	//删除文档
	if err := o.DoTx(func(ctx context.Context, txOrm orm.TxOrmer) error {
		_, err = txOrm.Raw("DELETE FROM "+NewDocument().TableNameWithPrefix()+" WHERE book_id = ?", book.BookId).Exec()
		return err
	}); err != nil {
		return err
	}
	//删除项目
	if err := o.DoTx(func(ctx context.Context, txOrm orm.TxOrmer) error {
		_, err = txOrm.Raw("DELETE FROM "+book.TableNameWithPrefix()+" WHERE book_id = ?", book.BookId).Exec()
		return err
	}); err != nil {
		return err
	}

	//删除关系

	if err := o.DoTx(func(ctx context.Context, txOrm orm.TxOrmer) error {
		_, err = txOrm.Raw("DELETE FROM "+NewRelationship().TableNameWithPrefix()+" WHERE book_id = ?", book.BookId).Exec()
		return err
	}); err != nil {
		return err
	}

	if err := o.DoTx(func(ctx context.Context, txOrm orm.TxOrmer) error {
		_, err = txOrm.Raw(fmt.Sprintf("DELETE FROM %s WHERE book_id=?", NewTeamRelationship().TableNameWithPrefix()), book.BookId).Exec()
		return err
	}); err != nil {
		return err
	}
	//删除模板

	if err := o.DoTx(func(ctx context.Context, txOrm orm.TxOrmer) error {
		_, err = txOrm.Raw("DELETE FROM "+NewTemplate().TableNameWithPrefix()+" WHERE book_id = ?", book.BookId).Exec()
		return err
	}); err != nil {
		return err
	}

	if book.Label != "" {
		NewLabel().InsertOrUpdateMulti(book.Label)
	}

	//删除导出缓存
	if err := os.RemoveAll(filepath.Join(config.GetExportOutputPath(), strconv.Itoa(id))); err != nil {
		logs.Error("删除项目缓存失败 ->", err)
	}
	//删除附件和图片
	if err := os.RemoveAll(filepath.Join(config.WorkingDirectory, "uploads", book.Identify)); err != nil {
		logs.Error("删除项目附件和图片失败 ->", err)
	}

	return nil

}

// ReleaseContent 批量发布文档
func (book *Book) ReleaseContent(bookId int, lang string) {
	releaseQueue <- bookId
	once.Do(func() {
		go func() {
			defer func() {
				if err := recover(); err != nil {
					logs.Error("协程崩溃 ->", err)
				}
			}()
			for bookId := range releaseQueue {
				o := orm.NewOrm()

				var docs []*Document
				_, err := o.QueryTable(NewDocument().TableNameWithPrefix()).Filter("book_id", bookId).All(&docs)

				if err != nil {
					logs.Error("发布失败 =>", bookId, err)
					continue
				}
				for _, item := range docs {
					item.BookId = bookId
					item.Lang = lang
					_ = item.ReleaseContent()
				}

				//当文档发布后，需要删除已缓存的转换项目
				outputPath := filepath.Join(config.GetExportOutputPath(), strconv.Itoa(bookId))
				_ = os.RemoveAll(outputPath)
			}
		}()
	})
}

// 重置文档数量
func (book *Book) ResetDocumentNumber(bookId int) {
	o := orm.NewOrm()

	totalCount, err := o.QueryTable(NewDocument().TableNameWithPrefix()).Filter("book_id", bookId).Count()
	if err == nil {
		_, err = o.Raw(fmt.Sprintf("UPDATE %s SET doc_count = ? WHERE book_id = ?", book.TableNameWithPrefix()), int(totalCount), bookId).Exec()
		if err != nil {
			logs.Error("重置文档数量失败 =>", bookId, err)
		}
	} else {
		logs.Error("获取文档数量失败 =>", bookId, err)
	}
}
