package auth

import "git.itopcms.com/jackliu/doc/internal/model"

// MemberIDFromSession extracts member id from session value.
// T6 stores int; legacy sessions may still hold Member / *Member.
func MemberIDFromSession(v any) (int, bool) {
	switch id := v.(type) {
	case int:
		return id, id > 0
	case int64:
		return int(id), id > 0
	case float64: // some session providers store numbers as float64
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
