package model

import (
	"fmt"

	"git.itopcms.com/astrueus/doc/internal/config"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

// 根据指定字段查询结果集.
func (book *Book) FindByField(field string, value any, cols ...string) ([]*Book, error) {
	o := orm.NewOrm()

	var books []*Book
	_, err := o.QueryTable(book.TableNameWithPrefix()).Filter(field, value).All(&books, cols...)

	return books, err
}

// 根据指定字段查询一个结果.
func (book *Book) FindByFieldFirst(field string, value any) (*Book, error) {
	o := orm.NewOrm()

	err := o.QueryTable(book.TableNameWithPrefix()).Filter(field, value).One(book)

	return book, err

}

// 根据项目标识查询项目
func (book *Book) FindByIdentify(identify string, cols ...string) (*Book, error) {
	o := orm.NewOrm()

	err := o.QueryTable(book.TableNameWithPrefix()).Filter("identify", identify).One(book, cols...)

	return book, err
}

func (book *Book) FindForRoleId(bookId, memberId int) (config.BookRole, error) {
	o := orm.NewOrm()

	var relationship Relationship

	err := NewRelationship().QueryTable().Filter("book_id", bookId).Filter("member_id", memberId).One(&relationship)

	if err != nil && err != orm.ErrNoRows {
		return 0, err
	}
	if err == nil {
		return relationship.RoleId, nil
	}
	sql := fmt.Sprintf(`select role_id
from %steam_relationship as mtr
left join %steam_member as mtm using (team_id)
where mtr.book_id = ? and mtm.member_id = ? order by mtm.role_id asc limit 1;`, config.GetDatabasePrefix(), config.GetDatabasePrefix())

	var roleId int
	err = o.Raw(sql, bookId, memberId).QueryRow(&roleId)

	if err != nil {
		logs.Error("查询用户项目角色出错 -> book_id=", bookId, " member_id=", memberId, err)
		return 0, err
	}
	return config.BookRole(roleId), nil
}
