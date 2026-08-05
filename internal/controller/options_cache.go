package controller

import (
	"sync/atomic"
	"time"

	"git.itopcms.com/astrueus/doc/internal/model"
	"github.com/beego/beego/v2/core/logs"
)

const optionsCacheTTL = 5 * time.Minute

var (
	optionsCache     atomic.Pointer[map[string]string]
	optionsExpiresAt atomic.Int64
)

// loadOptions returns site options with a short in-memory cache.
func loadOptions() map[string]string {
	now := time.Now().Unix()
	if p := optionsCache.Load(); p != nil && now < optionsExpiresAt.Load() {
		return *p
	}

	options, err := model.NewOption().All()
	if err != nil {
		logs.Error("加载 options 失败 ->", err)
		if p := optionsCache.Load(); p != nil {
			return *p
		}
		return map[string]string{}
	}

	m := make(map[string]string, len(options))
	for _, item := range options {
		m[item.OptionName] = item.OptionValue
	}
	optionsCache.Store(&m)
	optionsExpiresAt.Store(time.Now().Add(optionsCacheTTL).Unix())
	return m
}

// InvalidateOptions clears the options cache so the next request reloads from DB.
func InvalidateOptions() {
	optionsExpiresAt.Store(0)
}
