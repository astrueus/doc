package model

import (
	"fmt"
	"time"

	"strings"

	"git.itopcms.com/astrueus/doc/internal/config"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

type DocumentSearchResult struct {
	DocumentId   int    `json:"doc_id"`
	DocumentName string `json:"doc_name"`
	// Identify 文档唯一标识
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

func NewDocumentSearchResult() *DocumentSearchResult {
	return &DocumentSearchResult{}
}

// 分页全局搜索.
func (m *DocumentSearchResult) FindToPager(keyword string, pageIndex, pageSize, memberId int) (searchResult []*DocumentSearchResult, totalCount int, err error) {
	o := orm.NewOrm()

	offset := (pageIndex - 1) * pageSize

	keyword = "%" + strings.Replace(keyword, " ", "%", -1) + "%"
	p := config.GetDatabasePrefix()

	if memberId <= 0 {
		sql1 := fmt.Sprintf(`SELECT count(doc.document_id) as total_count FROM %sdocuments AS doc
  LEFT JOIN %sbooks as book ON doc.book_id = book.book_id
WHERE book.privately_owned = 0 AND (doc.document_name LIKE ? OR doc.release LIKE ?) `, p, p)

		sql2 := fmt.Sprintf(`SELECT *
FROM (
       SELECT
         doc.document_id,
         doc.modify_time,
         doc.create_time,
         doc.document_name,
         doc.identify,
         doc.release    AS description,
         book.identify  AS book_identify,
         book.book_name,
         rel.member_id,
         member.account AS author,
         'document'     AS search_type
       FROM %sdocuments AS doc
         LEFT JOIN %sbooks AS book ON doc.book_id = book.book_id
         LEFT JOIN %srelationship AS rel ON book.book_id = rel.book_id AND rel.role_id = 0
         LEFT JOIN %smembers AS member ON rel.member_id = member.member_id
       WHERE book.privately_owned = 0 AND (doc.document_name LIKE ? OR doc.release LIKE ?)
     UNION ALL
SELECT
  book.book_id AS document_id,
  book.modify_time,
  book.create_time,
  book.book_name AS document_name,
  book.identify,
  book.description,
  book.identify  AS book_identify,
  book.book_name,
  rel.member_id,
  member.account AS author,
  'book'     AS search_type
FROM  %sbooks AS book
       LEFT JOIN %srelationship AS rel ON book.book_id = rel.book_id AND rel.role_id = 0
       LEFT JOIN %smembers AS member ON rel.member_id = member.member_id
WHERE book.privately_owned = 0 AND (book.book_name LIKE ? OR book.description LIKE ?)

       UNION ALL
       SELECT
         blog.blog_id AS document_id,
         blog.modify_time,
         blog.create_time,
         blog.blog_title as document_name,
         blog.blog_identify,
         blog.blog_release,
         blog.blog_identify,
         blog.blog_title as book_name,
         blog.member_id,
         member.account,
         'blog' AS search_type
       FROM %sblogs AS blog
         LEFT JOIN %smembers AS member ON blog.member_id = member.member_id
       WHERE blog.blog_status = 'public' AND (blog.blog_release LIKE ? OR blog.blog_title LIKE ?)
     ) AS union_table
ORDER BY create_time DESC
LIMIT ?, ?;`, p, p, p, p, p, p, p, p, p)

		err = o.Raw(sql1, keyword, keyword).QueryRow(&totalCount)
		if err != nil {
			logs.Error("查询搜索结果失败 -> ", err)
			return
		}
		sql3 := fmt.Sprintf(`       SELECT
         count(*)
       FROM %sblogs AS blog
       WHERE blog.blog_status = 'public' AND (blog.blog_release LIKE ? OR blog.blog_title LIKE ?);`, p)

		c := 0
		err = o.Raw(sql3, keyword, keyword).QueryRow(&c)
		if err != nil {
			logs.Error("查询搜索结果失败 -> ", err)
			return
		}

		totalCount += c
		//查询项目的数量
		sql4 := fmt.Sprintf(`SELECT count(*) as total_count FROM %sbooks as book
WHERE book.privately_owned = 0 AND (book.book_name LIKE ? OR book.description LIKE ?);`, p)

		err = o.Raw(sql4, keyword, keyword).QueryRow(&c)
		if err != nil {
			logs.Error("查询搜索结果失败 -> ", err)
			return
		}

		totalCount += c

		_, err = o.Raw(sql2, keyword, keyword, keyword, keyword, keyword, keyword, offset, pageSize).QueryRows(&searchResult)
		if err != nil {
			logs.Error("查询搜索结果失败 -> ", err)
			return
		}
	} else {
		sql1 := fmt.Sprintf(`SELECT count(doc.document_id) as total_count FROM %sdocuments AS doc
  LEFT JOIN %sbooks as book ON doc.book_id = book.book_id
  LEFT JOIN %srelationship AS rel ON doc.book_id = rel.book_id AND rel.role_id = 0
  LEFT JOIN %srelationship AS rel1 ON doc.book_id = rel1.book_id AND rel1.member_id = ?
			left join (select * from (select book_id,team_member_id,role_id
                   	from %steam_relationship as mtr
					left join %steam_member as mtm on mtm.team_id=mtr.team_id and mtm.member_id=? order by role_id desc )as t group by t.role_id,t.team_member_id,t.book_id) as team 
					on team.book_id = book.book_id
WHERE (book.privately_owned = 0 OR rel1.relationship_id > 0 or team.team_member_id > 0)  AND (doc.document_name LIKE ? OR doc.release LIKE ?);`, p, p, p, p, p, p)

		sql2 := fmt.Sprintf(`SELECT *
FROM (
       SELECT
         doc.document_id,
         doc.modify_time,
         doc.create_time,
         doc.document_name,
         doc.identify,
         doc.release    AS description,
         book.identify  AS book_identify,
         book.book_name,
         rel.member_id,
         member.account AS author,
         'document'     AS search_type
       FROM %sdocuments AS doc
         LEFT JOIN %sbooks AS book ON doc.book_id = book.book_id
         LEFT JOIN %srelationship AS rel ON book.book_id = rel.book_id AND rel.role_id = 0
         LEFT JOIN %smembers AS member ON rel.member_id = member.member_id
         LEFT JOIN %srelationship AS rel1 ON doc.book_id = rel1.book_id AND rel1.member_id = ?
         LEFT JOIN (SELECT *
                    FROM (SELECT
                            book_id,
                            team_member_id,
                            role_id
                          FROM %steam_relationship AS mtr
                            LEFT JOIN %steam_member AS mtm ON mtm.team_id = mtr.team_id AND mtm.member_id = ?
                          ORDER BY role_id DESC) AS t
                    GROUP BY t.role_id, t.team_member_id, t.book_id) AS team
           ON team.book_id = book.book_id
       WHERE (book.privately_owned = 0 OR rel1.relationship_id > 0 OR team.team_member_id > 0) AND
             (doc.document_name LIKE ? OR doc.release LIKE ?)
       UNION ALL

       SELECT
         book.book_id AS document_id,
         book.modify_time,
         book.create_time,
         book.book_name AS document_name,
         book.identify,
         book.description AS description,
         book.identify  AS book_identify,
         book.book_name,
         rel.member_id,
         member.account AS author,
         'book'     AS search_type
       FROM %sbooks AS book
         LEFT JOIN %srelationship AS rel ON book.book_id = rel.book_id AND rel.role_id = 0
         LEFT JOIN %smembers AS member ON rel.member_id = member.member_id
         LEFT JOIN %srelationship AS rel1 ON book.book_id = rel1.book_id AND rel1.member_id = ?
         LEFT JOIN (SELECT *
                    FROM (SELECT
                            book_id,
                            team_member_id,
                            role_id
                          FROM %steam_relationship AS mtr
                            LEFT JOIN %steam_member AS mtm ON mtm.team_id = mtr.team_id AND mtm.member_id = ?
                          ORDER BY role_id DESC) AS t
                    GROUP BY t.role_id, t.team_member_id, t.book_id) AS team
           ON team.book_id = book.book_id
       WHERE (book.privately_owned = 0 OR rel1.relationship_id > 0 OR team.team_member_id > 0) AND
             (book.book_name LIKE ? OR book.description LIKE ?)
 UNION ALL
       SELECT
         blog.blog_id AS document_id, 
         blog.modify_time,
         blog.create_time,
         blog.blog_title as document_name,
         blog.blog_identify as identify,
         blog.blog_release as description,
         blog.blog_identify  AS book_identify,
         blog.blog_title as book_name,
         blog.member_id,
         member.account,
         'blog' AS search_type
       FROM %sblogs AS blog
         LEFT JOIN %smembers AS member ON blog.member_id = member.member_id
       WHERE (blog.blog_status = 'public' OR blog.member_id = ?) AND blog.blog_type = 0 AND
             (blog.blog_release LIKE ? OR blog.blog_title LIKE ?)
     ) AS union_table
ORDER BY create_time DESC
LIMIT ?, ?;`, p, p, p, p, p, p, p, p, p, p, p, p, p, p, p)

		err = o.Raw(sql1, memberId, memberId, keyword, keyword).QueryRow(&totalCount)
		if err != nil {
			return
		}
		sql3 := fmt.Sprintf(`       SELECT
         count(*)
       FROM %sblogs AS blog
       WHERE (blog.blog_status = 'public' OR blog.member_id = ?) AND blog.blog_type = 0 AND
             (blog.blog_release LIKE ? OR blog.blog_title LIKE ?);`, p)

		c := 0
		err = o.Raw(sql3, memberId, keyword, keyword).QueryRow(&c)
		if err != nil {
			logs.Error("查询搜索结果失败 -> ", err)
			return
		}

		totalCount += c

		sql4 := fmt.Sprintf(`SELECT count(*) as total_count FROM %sbooks as book
  LEFT JOIN %srelationship AS rel ON book.book_id = rel.book_id AND rel.role_id = 0
  LEFT JOIN %srelationship AS rel1 ON book.book_id = rel1.book_id AND rel1.member_id = ?
			left join (select * from (select book_id,team_member_id,role_id
                   	from %steam_relationship as mtr
					left join %steam_member as mtm on mtm.team_id=mtr.team_id and mtm.member_id=? order by role_id desc )as t group by t.role_id,t.team_member_id,t.book_id) as team
					on team.book_id = book.book_id
WHERE (book.privately_owned = 0 OR rel1.relationship_id > 0 or team.team_member_id > 0)  AND (book.book_name LIKE ? OR book.description LIKE ?);`, p, p, p, p, p)

		err = o.Raw(sql4, memberId, memberId, keyword, keyword).QueryRow(&c)
		if err != nil {
			logs.Error("查询搜索结果失败 -> ", err)
			return
		}

		totalCount += c

		_, err = o.Raw(sql2, memberId, memberId, keyword, keyword, memberId, memberId, keyword, keyword, memberId, keyword, keyword, offset, pageSize).QueryRows(&searchResult)
		if err != nil {
			return
		}
	}
	return
}

// 项目内搜索.
func (m *DocumentSearchResult) SearchDocument(keyword string, bookId int) (docs []*DocumentSearchResult, err error) {
	o := orm.NewOrm()

	sql := fmt.Sprintf("SELECT * FROM %s WHERE book_id = ? AND (document_name LIKE ? OR `release` LIKE ?) ", NewDocument().TableNameWithPrefix())
	keyword = "%" + keyword + "%"

	_, err = o.Raw(sql, bookId, keyword, keyword).QueryRows(&docs)

	return
}
