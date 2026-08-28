package model

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

// 复制项目
func (book *Book) Copy(identify string) error {
	o := orm.NewOrm()

	err := o.QueryTable(book.TableNameWithPrefix()).Filter("identify", identify).One(book)

	if err != nil {
		logs.Error("查询项目时出错 -> ", err)
		return err
	}
	if _, err := o.Begin(); err != nil {
		logs.Error("开启事物时出错 -> ", err)
		return err
	}

	bookId := book.BookId
	book.BookId = 0
	book.Identify = book.Identify + fmt.Sprintf("%s-%s", identify, strconv.FormatInt(time.Now().UnixNano(), 32))
	book.BookName = book.BookName + "[副本]"
	book.CreateTime = time.Now()
	book.CommentCount = 0
	book.HistoryCount = 0

	/* v2 version of beego remove the o.Rollback api for transaction operation.
	 * typically, in v1, you can write code like this:
	 *
	 *		o := orm.NewOrm()
	 *		if err := o.Operateion(); err != nil {
	 *			o.Rollback()
	 *			...
	 *		}
	 *
	 * however, in v2, this is not available. beego will handles the transaction in new way using
	 * cluster. the new code is like below:
	 *
	 * 		o := orm.NewOrm()
	 * 		if err := o.DoTx(func(ctx context.Context, txOrm orm.TxOrmer) error{
	 *			err := o.Operations()
	 *			if err != nil {
	 *				return err
	 * 			}
	 *			...
	 * 		}); err != nil {
	 *			...
	 * 		}
	 *
	 * 	when operation failed, it will automatically calls o.Rollback() for TxOrmer.
	 *  more details see https://beego.me/docs/mvc/model/transaction.md
	 */
	if err := o.DoTx(func(ctx context.Context, txo orm.TxOrmer) error {
		_, err := txo.Insert(book)
		return err

	}); err != nil {
		logs.Error("复制项目时出错： %s", err)
		return err
	}

	var rels []*Relationship

	if err := o.DoTx(func(ctx context.Context, txo orm.TxOrmer) error {
		_, err := txo.QueryTable(NewRelationship().TableNameWithPrefix()).Filter("book_id", bookId).All(&rels)
		return err
	}); err != nil {
		logs.Error("复制项目关系时出错 -> ", err)
		return err
	}

	for _, rel := range rels {
		rel.BookId = book.BookId
		rel.RelationshipId = 0
		if err := o.DoTx(func(ctx context.Context, txo orm.TxOrmer) error {
			_, err := txo.Insert(rel)
			return err
		}); err != nil {
			logs.Error("复制项目关系时出错 -> ", err)
			return err
		}
	}

	var docs []*Document

	if err := o.DoTx(func(ctx context.Context, txOrm orm.TxOrmer) error {
		_, err := txOrm.QueryTable(NewDocument().TableNameWithPrefix()).Filter("book_id", bookId).Filter("parent_id", 0).All(&docs)
		return err
	}); err != nil && err != orm.ErrNoRows {
		logs.Error("读取项目文档时出错 -> ", err)
		return err
	}

	if len(docs) > 0 {
		if err := o.DoTx(func(ctx context.Context, txOrm orm.TxOrmer) error {
			return recursiveInsertDocument(docs, txOrm, book.BookId, 0)
		}); err != nil {
			logs.Error("复制项目时出错 -> ", err)
			return err
		}
	}

	return nil
}

// 递归的复制文档
func recursiveInsertDocument(docs []*Document, o orm.TxOrmer, bookId int, parentId int) error {
	for _, doc := range docs {

		docId := doc.DocumentId
		doc.DocumentId = 0
		doc.ParentId = parentId
		doc.BookId = bookId
		doc.Version = time.Now().Unix()

		if _, err := o.Insert(doc); err != nil {
			logs.Error("插入项目时出错 -> ", err)
			return err
		}

		var attachList []*Attachment
		//读取所有附件列表
		if _, err := o.QueryTable(NewAttachment().TableNameWithPrefix()).Filter("document_id", docId).All(&attachList); err == nil {
			for _, attach := range attachList {
				attach.BookId = bookId
				attach.DocumentId = doc.DocumentId
				attach.AttachmentId = 0
				if _, err := o.Insert(attach); err != nil {
					return err
				}
			}
		}
		var subDocs []*Document

		if _, err := o.QueryTable(NewDocument().TableNameWithPrefix()).Filter("parent_id", docId).All(&subDocs); err != nil && err != orm.ErrNoRows {
			logs.Error("读取文档时出错 -> ", err)
			return err
		}
		if len(subDocs) > 0 {

			if err := recursiveInsertDocument(subDocs, o, bookId, doc.DocumentId); err != nil {
				return err
			}
		}
	}
	return nil
}
