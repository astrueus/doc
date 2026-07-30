package config

// Session / captcha
const LoginSessionName = "LoginSessionName"

const CaptchaSessionName = "__captcha__"

const RegexpEmail = "^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$"

// 允许用户名中出现点号
const RegexpAccount = `^[a-zA-Z][a-zA-Z0-9\.-]{2,50}$`

// PageSize 默认分页条数.
const PageSize = 10

// 用户权限
const (
	// 超级管理员.
	MemberSuperRole SystemRole = iota
	//普通管理员.
	MemberAdminRole
	//普通用户.
	MemberGeneralRole
)

// SystemRole 系统角色
type SystemRole int

const (
	// 创始人.
	BookFounder BookRole = iota
	//管理者
	BookAdmin
	//编辑者.
	BookEditor
	//观察者
	BookObserver
)

// BookRole 项目角色
type BookRole int

const (
	LoggerOperate   = "operate"
	LoggerSystem    = "system"
	LoggerException = "exception"
	LoggerDocument  = "document"
)

const (
	//本地账户校验
	AuthMethodLocal = "local"
	//LDAP用户校验
	AuthMethodLDAP = "ldap"
)

// Build-time metadata (ldflags).
var (
	VERSION    string
	BUILD_TIME string
	GO_VERSION string
)
