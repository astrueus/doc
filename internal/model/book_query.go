package model

import (
	"fmt"
	"time"

	"git.itopcms.com/jackliu/doc/internal/config"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"git.itopcms.com/jackliu/doc/internal/i18n"
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

// 分页查询指定用户的项目
func (book *Book) FindToPager(pageIndex, pageSize, memberId int, lang string) (books []*BookResult, totalCount int, err error) {

	o := orm.NewOrm()

	p := config.GetDatabasePrefix()

	//sql1 := "SELECT COUNT(book.book_id) AS total_count FROM " + book.TableNameWithPrefix() + " AS book LEFT JOIN " +
	//	relationship.TableNameWithPrefix() + " AS rel ON book.book_id=rel.book_id AND rel.member_id = ? WHERE rel.relationship_id > 0 "

	sql1 := fmt.Sprintf(`SELECT
count(*) AS total_count
FROM %sbooks AS book
  LEFT JOIN %srelationship AS rel ON book.book_id = rel.book_id AND rel.member_id = ?
  left join (select book_id,min(role_id) as role_id
             from (select book_id,team_member_id,role_id
                   from %steam_relationship as mtr
                     left join %steam_member as mtm on mtm.team_id=mtr.team_id and mtm.member_id=? order by role_id desc )
					as t group by t.book_id)
			as team on team.book_id=book.book_id WHERE rel.role_id >= 0 or team.role_id >= 0`, p, p, p, p)

	err = o.Raw(sql1, memberId, memberId).QueryRow(&totalCount)

	if err != nil {
		return
	}

	offset := (pageIndex - 1) * pageSize

	//sql2 := "SELECT book.*,rel.member_id,rel.role_id,m.account as create_name FROM " + book.TableNameWithPrefix() + " AS book" +
	//	" LEFT JOIN " + relationship.TableNameWithPrefix() + " AS rel ON book.book_id=rel.book_id AND rel.member_id = ?" +
	//	" LEFT JOIN " + relationship.TableNameWithPrefix() + " AS rel1 ON book.book_id=rel1.book_id  AND rel1.role_id=0" +
	//	" LEFT JOIN " + NewMember().TableNameWithPrefix() + " AS m ON rel1.member_id=m.member_id " +
	//	" WHERE rel.relationship_id > 0 ORDER BY book.order_index DESC,book.book_id DESC LIMIT " + fmt.Sprintf("%d,%d", offset, pageSize)

	sql2 := fmt.Sprintf(`SELECT
  book.*,
  case when rel.relationship_id  is null then team.role_id else rel.role_id end as role_id,
  m.account as create_name
FROM %sbooks AS book
  LEFT JOIN %srelationship AS rel ON book.book_id = rel.book_id AND rel.member_id = ?
  left join (select book_id,min(role_id) as role_id
             from (select book_id,team_member_id,role_id
                   from %steam_relationship as mtr
                     left join %steam_member as mtm on mtm.team_id=mtr.team_id and mtm.member_id=? order by role_id desc )
					as t group by book_id) as team 
			on team.book_id=book.book_id
  LEFT JOIN %srelationship AS rel1 ON book.book_id = rel1.book_id AND rel1.role_id = 0
  LEFT JOIN %smembers AS m ON rel1.member_id = m.member_id
WHERE rel.role_id >= 0 or team.role_id >= 0
ORDER BY book.order_index, book.book_id DESC limit ?,?`, p, p, p, p, p, p)

	_, err = o.Raw(sql2, memberId, memberId, offset, pageSize).QueryRows(&books)
	if err != nil {
		logs.Error("分页查询项目列表 => ", err)
		return
	}
	sql := fmt.Sprintf("SELECT m.account,doc.modify_time FROM %sdocuments AS doc LEFT JOIN %smembers AS m ON doc.modify_at=m.member_id WHERE book_id = ? LIMIT 1 ORDER BY doc.modify_time DESC", p, p)

	if len(books) > 0 {
		for index, book := range books {
			var text struct {
				Account    string
				ModifyTime time.Time
			}

			err1 := o.Raw(sql, book.BookId).QueryRow(&text)
			if err1 == nil {
				books[index].LastModifyText = text.Account + " 于 " + text.ModifyTime.Format("2006-01-02 15:04:05")
			}
			if book.RoleId == 0 {
				book.RoleName = i18n.Tr(lang, "common.creator")
			} else if book.RoleId == 1 {
				book.RoleName = i18n.Tr(lang, "common.administrator")
			} else if book.RoleId == 2 {
				book.RoleName = i18n.Tr(lang, "common.editor")
			} else if book.RoleId == 3 {
				book.RoleName = i18n.Tr(lang, "common.observer")
			}
		}
	}
	return
}


// 分页查找系统首页数据.
func (book *Book) FindForHomeToPager(pageIndex, pageSize, memberId int) (books []*BookResult, totalCount int, err error) {
	o := orm.NewOrm()

	offset := (pageIndex - 1) * pageSize
	p := config.GetDatabasePrefix()
	//如果是登录用户
	if memberId > 0 {
		sql1 := fmt.Sprintf(`SELECT COUNT(*)
FROM %sbooks AS book
  LEFT JOIN %srelationship AS rel ON rel.book_id = book.book_id AND rel.member_id = ?
  left join (select book_id,min(role_id) AS role_id
             from (select book_id,role_id
                   from %steam_relationship as mtr
                     left join %steam_member as mtm on mtm.team_id=mtr.team_id and mtm.member_id=? order by role_id desc )
as t group by book_id) as team on team.book_id=book.book_id
WHERE book.privately_owned = 0 or rel.role_id >=0 or team.role_id >=0`, p, p, p, p)
		err = o.Raw(sql1, memberId, memberId).QueryRow(&totalCount)
		if err != nil {
			return
		}
		sql2 := fmt.Sprintf(`SELECT book.*,rel1.*,member.account AS create_name,member.real_name FROM %sbooks AS book
  LEFT JOIN %srelationship AS rel ON rel.book_id = book.book_id AND rel.member_id = ?
  left join (select book_id,min(role_id) AS role_id
             from (select book_id,role_id
                   from %steam_relationship as mtr
                     left join %steam_member as mtm on mtm.team_id=mtr.team_id and mtm.member_id=? order by role_id desc )
as t group by book_id) as team on team.book_id=book.book_id
  LEFT JOIN %srelationship AS rel1 ON rel1.book_id = book.book_id AND rel1.role_id = 0
  LEFT JOIN %smembers AS member ON rel1.member_id = member.member_id
WHERE book.privately_owned = 0 or rel.role_id >=0 or team.role_id >=0 ORDER BY order_index desc,book.book_id DESC LIMIT ?,?`, p, p, p, p, p, p)

		_, err = o.Raw(sql2, memberId, memberId, offset, pageSize).QueryRows(&books)

	} else {
		count, err1 := o.QueryTable(book.TableNameWithPrefix()).Filter("privately_owned", 0).Count()

		if err1 != nil {
			err = err1
			return
		}
		totalCount = int(count)

		sql := fmt.Sprintf(`SELECT book.*,rel.*,member.account AS create_name,member.real_name FROM %sbooks AS book
			LEFT JOIN %srelationship AS rel ON rel.book_id = book.book_id AND rel.role_id = 0
			LEFT JOIN %smembers AS member ON rel.member_id = member.member_id
			WHERE book.privately_owned = 0 ORDER BY order_index DESC ,book.book_id DESC LIMIT ?,?`, p, p, p)

		_, err = o.Raw(sql, offset, pageSize).QueryRows(&books)

	}
	return
}

// 分页全局搜索.
func (book *Book) FindForLabelToPager(keyword string, pageIndex, pageSize, memberId int) (books []*BookResult, totalCount int, err error) {
	o := orm.NewOrm()

	keyword = "%" + keyword + "%"
	offset := (pageIndex - 1) * pageSize
	p := config.GetDatabasePrefix()
	//如果是登录用户
	if memberId > 0 {
		sql1 := fmt.Sprintf(`SELECT COUNT(*)
FROM %sbooks AS book
  LEFT JOIN %srelationship AS rel ON rel.book_id = book.book_id AND rel.member_id = ?
  left join (select *
             from (select book_id,team_member_id,role_id
                   from %steam_relationship as mtr
                     left join %steam_member as mtm on mtm.team_id=mtr.team_id and mtm.member_id=? order by role_id desc )as t group by t.role_id,t.team_member_id,t.book_id) as team on team.book_id = book.book_id
WHERE (relationship_id > 0 OR book.privately_owned = 0 or team.team_member_id > 0) AND book.label LIKE ?`, p, p, p, p)

		err = o.Raw(sql1, memberId, memberId, keyword).QueryRow(&totalCount)
		if err != nil {
			return
		}
		sql2 := fmt.Sprintf(`SELECT book.*,rel1.*,member.account AS create_name FROM %sbooks AS book
			LEFT JOIN %srelationship AS rel ON rel.book_id = book.book_id AND rel.member_id = ?
			left join (select * from (select book_id,team_member_id,role_id
                   	from %steam_relationship as mtr
					left join %steam_member as mtm on mtm.team_id=mtr.team_id and mtm.member_id=? order by role_id desc )as t group by t.role_id,t.team_member_id,t.book_id) as team 
					on team.book_id = book.book_id
			LEFT JOIN %srelationship AS rel1 ON rel1.book_id = book.book_id AND rel1.role_id = 0
			LEFT JOIN %smembers AS member ON rel1.member_id = member.member_id
			WHERE (rel.relationship_id > 0 OR book.privately_owned = 0 or team.team_member_id > 0) 
			AND book.label LIKE ? ORDER BY order_index DESC ,book.book_id DESC LIMIT ?,?`, p, p, p, p, p, p)

		_, err = o.Raw(sql2, memberId, memberId, keyword, offset, pageSize).QueryRows(&books)

		return

	} else {
		count, err1 := o.QueryTable(NewBook().TableNameWithPrefix()).Filter("privately_owned", 0).Filter("label__icontains", keyword).Count()

		if err1 != nil {
			err = err1
			return
		}
		totalCount = int(count)

		sql := fmt.Sprintf(`SELECT book.*,rel.*,member.account AS create_name FROM %sbooks AS book
			LEFT JOIN %srelationship AS rel ON rel.book_id = book.book_id AND rel.role_id = 0
			LEFT JOIN %smembers AS member ON rel.member_id = member.member_id
			WHERE book.privately_owned = 0 AND book.label LIKE ? ORDER BY order_index DESC ,book.book_id DESC LIMIT ?,?`, p, p, p)

		_, err = o.Raw(sql, keyword, offset, pageSize).QueryRows(&books)

		return

	}
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
