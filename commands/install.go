package commands

import (
	"errors"
	"fmt"
	"os"
	"time"

	"git.itopcms.com/jackliu/doc/conf"
	"git.itopcms.com/jackliu/doc/models"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/i18n"
)

// Install 系统安装.
func Install() {

	fmt.Println("Initializing...")

	err := orm.RunSyncdb("default", false, true)
	if err == nil {
		initialization()
	} else {
		panic(err.Error())
	}
	fmt.Println("Install Successfully!")
	os.Exit(0)

}

// 初始化数据
func initialization() {

	err := models.NewOption().Init()
	if err != nil {
		panic(err.Error())
	}

	lang, _ := web.AppConfig.String("default_lang")
	err = i18n.SetMessage(lang, "configs/lang/"+lang+".ini")
	if err != nil {
		panic(fmt.Errorf("initialize locale error: %s", err))
	}

	member, err := models.NewMember().FindByFieldFirst("account", "admin")
	if errors.Is(err, orm.ErrNoRows) {

		// create admin user
		logs.Info("creating admin user")
		member.Account = "admin"
		member.Avatar = conf.URLForWithCdnImage("/static/images/headimgurl.jpg")
		member.Password = "123456"
		member.AuthMethod = "local"
		member.Role = conf.MemberSuperRole
		member.Email = "admin@iminho.me"

		if err := member.Add(); err != nil {
			panic("Member.Add => " + err.Error())
		}

		// create demo book
		logs.Info("creating demo book")
		book := models.NewBook()

		book.MemberId = member.MemberId
		book.BookName = i18n.Tr(lang, "init.default_proj_name") //"Doc演示项目"
		book.Status = 0
		book.ItemId = 1
		book.Description = i18n.Tr(lang, "init.default_proj_desc") //"这是一个Doc演示项目，该项目是由系统初始化时自动创建。"
		book.CommentCount = 0
		book.PrivatelyOwned = 0
		book.CommentStatus = "closed"
		book.Identify = "mindoc"
		book.DocCount = 0
		book.CommentCount = 0
		book.Version = time.Now().Unix()
		book.Cover = conf.GetDefaultCover()
		book.Editor = "markdown"
		book.Theme = "default"

		if err := book.Insert(lang); err != nil {
			panic("初始化项目失败 -> " + err.Error())
		}
	} else if err != nil {
		panic(fmt.Errorf("occur errors when initialize: %s", err))
	}

	if !models.NewItemsets().Exist(1) {
		item := models.NewItemsets()
		item.ItemName = i18n.Tr(lang, "init.default_proj_space") //"默认项目空间"
		item.MemberId = 1
		if err := item.Save(); err != nil {
			panic("初始化项目空间失败 -> " + err.Error())
		}
	}
}
