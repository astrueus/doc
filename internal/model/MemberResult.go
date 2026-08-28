package model

// SelectMemberResult 选择器 JSON（项目空间 / 团队等）。
// 暂留 model：TeamMember / Itemsets / TeamRelationship 的查询方法返回该类型，
// model 禁止 import dto。
type SelectMemberResult struct {
	Result []KeyValueItem `json:"results"`
}

// KeyValueItem 选择器条目。
type KeyValueItem struct {
	Id   int    `json:"id"`
	Text string `json:"text"`
}
