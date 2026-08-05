package auth

import "git.itopcms.com/astrueus/doc/internal/model"

// MemberIDFromSession 从 session 值中解析成员 ID。
// T6 起只存 int；旧 session 可能仍是 Member / *Member。
func MemberIDFromSession(v any) (int, bool) {
	switch id := v.(type) {
	case int:
		return id, id > 0
	case int64:
		return int(id), id > 0
	case float64: // 部分 session 驱动会把数字存成 float64
		return int(id), int(id) > 0
	case model.Member:
		return id.MemberId, id.MemberId > 0
	case *model.Member:
		if id != nil {
			return id.MemberId, id.MemberId > 0
		}
	}
	return 0, false
}
