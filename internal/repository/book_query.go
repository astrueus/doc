package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"git.itopcms.com/astrueus/doc/internal/config"
	"git.itopcms.com/astrueus/doc/internal/dto"
	"git.itopcms.com/astrueus/doc/internal/i18n"
	"git.itopcms.com/astrueus/doc/internal/model"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

func (r *bookRepo) FindToPagerForMember(ctx context.Context, pageIndex, pageSize, memberId int, lang string) (books []*dto.BookResult, totalCount int, err error) {
	o := r.ormer(ctx)

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
func (r *bookRepo) FindForHomeToPager(ctx context.Context, pageIndex, pageSize, memberId int) (books []*dto.BookResult, totalCount int, err error) {
	o := r.ormer(ctx)

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
		count, err1 := o.QueryTable(model.NewBook().TableNameWithPrefix()).Filter("privately_owned", 0).Count()

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
func (r *bookRepo) FindForLabelToPager(ctx context.Context, keyword string, pageIndex, pageSize, memberId int) (books []*dto.BookResult, totalCount int, err error) {
	o := r.ormer(ctx)

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
		count, err1 := o.QueryTable(model.NewBook().TableNameWithPrefix()).Filter("privately_owned", 0).Filter("label__icontains", keyword).Count()

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

func (r *bookRepo) FindForRoleId(ctx context.Context, bookId, memberId int) (config.BookRole, error) {
	o := r.ormer(ctx)

	var relationship model.Relationship

	err := model.NewRelationship().QueryTable().Filter("book_id", bookId).Filter("member_id", memberId).One(&relationship)

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

func (r *bookRepo) ListAllIDs(ctx context.Context) ([]int, error) {
	var ids []int
	_, err := r.ormer(ctx).Raw(fmt.Sprintf("SELECT book_id FROM %s ORDER BY book_id", model.NewBook().TableNameWithPrefix())).QueryRows(&ids)
	return ids, err
}

func (r *bookRepo) ListVisibleIDs(ctx context.Context, memberID int) ([]int, error) {
	o := r.ormer(ctx)
	var ids []int
	p := config.GetDatabasePrefix()
	_, err := o.Raw(fmt.Sprintf(`SELECT DISTINCT book.book_id
FROM %sbooks AS book
  LEFT JOIN %srelationship AS rel1 ON book.book_id = rel1.book_id AND rel1.member_id = ?
  LEFT JOIN (
    SELECT book_id, team_member_id
    FROM (
      SELECT book_id, team_member_id, role_id
      FROM %steam_relationship AS mtr
        LEFT JOIN %steam_member AS mtm ON mtm.team_id = mtr.team_id AND mtm.member_id = ?
      ORDER BY role_id DESC
    ) AS t
    GROUP BY t.role_id, t.team_member_id, t.book_id
  ) AS team ON team.book_id = book.book_id
WHERE book.privately_owned = 0 OR rel1.relationship_id > 0 OR team.team_member_id > 0`, p, p, p, p)).QueryRows(&ids)
	return ids, err
}

func (r *bookRepo) FindToPagerAll(ctx context.Context, pageIndex, pageSize int) (books []*dto.BookResult, totalCount int, err error) {
	o := r.ormer(ctx)
	count, err := o.QueryTable(model.NewBook().TableNameWithPrefix()).Count()
	if err != nil {
		return
	}
	totalCount = int(count)
	sql := fmt.Sprintf(`SELECT
			book.*,rel.relationship_id,rel.role_id,m.account AS create_name,m.real_name
		FROM %sbooks AS book
			LEFT JOIN %srelationship AS rel ON rel.book_id = book.book_id AND rel.role_id = 0
			LEFT JOIN %smembers AS m ON rel.member_id = m.member_id
		ORDER BY book.order_index DESC ,book.book_id DESC  LIMIT ?,?`, config.GetDatabasePrefix(), config.GetDatabasePrefix(), config.GetDatabasePrefix())
	offset := (pageIndex - 1) * pageSize
	_, err = o.Raw(sql, offset, pageSize).QueryRows(&books)
	return
}

func (r *bookRepo) FindByIdentifyForMember(ctx context.Context, identify string, memberID int, lang string) (*dto.BookResult, error) {
	if identify == "" || memberID <= 0 {
		return dto.NewBookResult(), model.ErrInvalidParameter
	}
	o := r.ormer(ctx)
	var book model.Book
	err := model.NewBook().QueryTable().Filter("identify", identify).One(&book)
	if err != nil {
		logs.Error("获取项目失败 ->", err)
		return dto.NewBookResult(), err
	}

	roleId, err := r.FindForRoleId(ctx, book.BookId, memberID)
	if err != nil {
		return dto.NewBookResult(), model.ErrPermissionDenied
	}
	var relationship2 model.Relationship
	err = model.NewRelationship().QueryTable().Filter("book_id", book.BookId).Filter("role_id", 0).One(&relationship2)
	if err != nil {
		logs.Error("根据项目标识查询项目以及指定用户权限的信息 -> ", err)
		return dto.NewBookResult(), model.ErrPermissionDenied
	}

	member, err := model.NewMember().Find(relationship2.MemberId)
	if err != nil {
		return dto.NewBookResult(), err
	}

	m := r.BookToResult(ctx, book)
	m.Lang = lang
	m.RoleId = roleId
	m.MemberId = memberID
	m.CreateName = member.Account
	if member.RealName != "" {
		m.RealName = member.RealName
	}
	applyBookRoleName(m, lang)
	fillLastModifyText(o, m)
	return m, nil
}

func (r *bookRepo) BookToResult(ctx context.Context, book model.Book) *dto.BookResult {
	m := dto.NewBookResult()
	m.BookId = book.BookId
	m.BookName = book.BookName
	m.Identify = book.Identify
	m.OrderIndex = book.OrderIndex
	m.Description = strings.Replace(book.Description, "\r\n", "<br/>", -1)
	m.PrivatelyOwned = book.PrivatelyOwned
	m.PrivateToken = book.PrivateToken
	m.BookPassword = book.BookPassword
	m.DocCount = book.DocCount
	m.CommentStatus = book.CommentStatus
	m.CommentCount = book.CommentCount
	m.CreateTime = book.CreateTime
	m.ModifyTime = book.ModifyTime
	m.Cover = book.Cover
	m.Label = book.Label
	m.Status = book.Status
	m.Editor = book.Editor
	m.Theme = book.Theme
	m.AutoRelease = book.AutoRelease == 1
	m.IsEnableShare = book.IsEnableShare == 0
	m.IsUseFirstDocument = book.IsUseFirstDocument == 1
	m.Publisher = book.Publisher
	m.HistoryCount = book.HistoryCount
	m.IsDownload = book.IsDownload == 0
	m.AutoSave = book.AutoSave == 1
	m.ItemId = book.ItemId
	if book.Theme == "" {
		m.Theme = "default"
	}
	if book.Editor == "" {
		m.Editor = "markdown"
	}
	fillLastModifyText(r.ormer(ctx), m)
	if m.ItemId > 0 {
		if item, err := model.NewItemsets().First(m.ItemId); err == nil {
			m.ItemName = item.ItemName
		}
	}
	return m
}

func fillLastModifyText(o orm.Ormer, m *dto.BookResult) {
	doc := model.NewDocument()
	err := o.QueryTable(doc.TableNameWithPrefix()).Filter("book_id", m.BookId).OrderBy("modify_time").One(doc)
	if err != nil {
		return
	}
	member2 := model.NewMember()
	member2.Find(doc.ModifyAt)
	m.LastModifyText = member2.Account + " 于 " + doc.ModifyTime.Local().Format("2006-01-02 15:04:05")
}

func applyBookRoleName(m *dto.BookResult, lang string) {
	if m.RoleId == config.BookFounder {
		m.RoleName = i18n.Tr(lang, "common.creator")
	} else if m.RoleId == config.BookAdmin {
		m.RoleName = i18n.Tr(lang, "common.administrator")
	} else if m.RoleId == config.BookEditor {
		m.RoleName = i18n.Tr(lang, "common.editor")
	} else if m.RoleId == config.BookObserver {
		m.RoleName = i18n.Tr(lang, "common.observer")
	}
}

func (r *bookRepo) FindToPagerByItemKey(ctx context.Context, key string, pageIndex, pageSize, memberID int) (books []*dto.BookResult, totalCount int, err error) {
	o := r.ormer(ctx)
	item := model.NewItemsets()
	err = item.QueryTable().Filter("item_key", key).One(item)
	if err != nil {
		logs.Error("查询项目空间时出错 ->", key, err)
		return nil, 0, err
	}
	offset := (pageIndex - 1) * pageSize
	p := config.GetDatabasePrefix()
	if memberID > 0 {
		sql1 := fmt.Sprintf(`SELECT COUNT(*)
FROM %sbooks AS book
  LEFT JOIN %srelationship AS rel ON rel.book_id = book.book_id AND rel.member_id = ?
  left join (select book_id,min(role_id) as role_id
             from (select book_id,role_id
                   from %steam_relationship as mtr
                     left join %steam_member as mtm on mtm.team_id=mtr.team_id and mtm.member_id=? order by role_id desc )
as t group by book_id) as team on team.book_id = book.book_id
WHERE book.item_id = ? AND (book.privately_owned = 0 or rel.role_id >= 0 or team.role_id >= 0)`, p, p, p, p)
		err = o.Raw(sql1, memberID, memberID, item.ItemId).QueryRow(&totalCount)
		if err != nil {
			logs.Error("查询项目空间时出错 ->", key, err)
			return
		}
		sql2 := fmt.Sprintf(`SELECT book.*,rel1.*,member.account AS create_name FROM %sbooks AS book
			LEFT JOIN %srelationship AS rel ON rel.book_id = book.book_id AND rel.member_id = ?
			left join (select book_id,min(role_id) as role_id from (select book_id,role_id
                   	from %steam_relationship as mtr
					left join %steam_member as mtm on mtm.team_id=mtr.team_id and mtm.member_id=? order by role_id desc )
as t group by book_id) as team 
					on team.book_id = book.book_id
			LEFT JOIN %srelationship AS rel1 ON rel1.book_id = book.book_id AND rel1.role_id = 0
			LEFT JOIN %smembers AS member ON rel1.member_id = member.member_id
			WHERE book.item_id = ? AND (book.privately_owned = 0 or rel.role_id >= 0 or team.role_id >= 0) 
			ORDER BY order_index desc,book.book_id DESC LIMIT ?,?`, p, p, p, p, p, p)
		_, err = o.Raw(sql2, memberID, memberID, item.ItemId, offset, pageSize).QueryRows(&books)
		return
	}
	count, err1 := o.QueryTable(model.NewBook().TableNameWithPrefix()).Filter("privately_owned", 0).Filter("item_id", item.ItemId).Count()
	if err1 != nil {
		err = err1
		return
	}
	totalCount = int(count)
	sql := fmt.Sprintf(`SELECT book.*,rel.*,member.account AS create_name FROM %sbooks AS book
			LEFT JOIN %srelationship AS rel ON rel.book_id = book.book_id AND rel.role_id = 0
			LEFT JOIN %smembers AS member ON rel.member_id = member.member_id
			WHERE book.item_id = ? AND book.privately_owned = 0 ORDER BY order_index desc,book.book_id DESC LIMIT ?,?`, p, p, p)
	_, err = o.Raw(sql, item.ItemId, offset, pageSize).QueryRows(&books)
	return
}
