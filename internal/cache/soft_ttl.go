package cache

import "time"

func softExpired(now, softAt time.Time) bool {
	if softAt.IsZero() {
		return false
	}
	return !now.Before(softAt)
}

func hardExpired(now, hardAt time.Time) bool {
	if hardAt.IsZero() {
		return false
	}
	return !now.Before(hardAt)
}
