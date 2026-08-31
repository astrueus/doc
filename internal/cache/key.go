package cache

import "strconv"

// DefaultKeyPrefix 为全新缓存前缀，不兼容旧 Document.Id.*。
const DefaultKeyPrefix = "doc:v1:"

// KeyBuilder 拼装稳定 key。
type KeyBuilder struct {
	Prefix string
}

// Keys 返回使用默认前缀的 KeyBuilder。
func Keys() KeyBuilder {
	return KeyBuilder{Prefix: DefaultKeyPrefix}
}

func (k KeyBuilder) prefix() string {
	if k.Prefix == "" {
		return DefaultKeyPrefix
	}
	return k.Prefix
}

// DocumentByID 文档主键。
func (k KeyBuilder) DocumentByID(id int) string {
	return k.prefix() + "document:id:" + strconv.Itoa(id)
}

// DocumentByIdentify 书内 identify。
func (k KeyBuilder) DocumentByIdentify(bookID int, identify string) string {
	return k.prefix() + "document:book:" + strconv.Itoa(bookID) + ":ident:" + identify
}

// BlogByID 博客主键。
func (k KeyBuilder) BlogByID(id int) string {
	return k.prefix() + "blog:id:" + strconv.Itoa(id)
}

// MCPToken Token 哈希 → Member。
func (k KeyBuilder) MCPToken(hash string) string {
	return k.prefix() + "mcp:token:" + hash
}

// TagBook 按书失效用 tag。
func (k KeyBuilder) TagBook(bookID int) string {
	return "book:" + strconv.Itoa(bookID)
}

// TagDocument 按文档失效用 tag。
func (k KeyBuilder) TagDocument(id int) string {
	return "document:" + strconv.Itoa(id)
}
