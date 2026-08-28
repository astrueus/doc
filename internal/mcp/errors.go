package mcp

import (
	"encoding/json"
	"strings"

	"git.itopcms.com/astrueus/doc/internal/errs"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type toolErrorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func toolBizError(err error) (*sdkmcp.CallToolResult, error) {
	b, ok := errs.AsBiz(err)
	if !ok {
		return nil, err
	}
	body, _ := json.Marshal(toolErrorBody{Code: b.Code, Message: b.Msg})
	return &sdkmcp.CallToolResult{
		IsError: true,
		Content: []sdkmcp.Content{
			&sdkmcp.TextContent{Text: string(body)},
		},
	}, nil
}

func toolBizErrorOut[T any](err error) (*sdkmcp.CallToolResult, T, error) {
	res, e := toolBizError(err)
	if e != nil {
		return nil, *new(T), e
	}
	return res, *new(T), nil
}

func defaultLimit(limit, def, max int) int {
	if limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}

func defaultPage(page int) int {
	if page <= 0 {
		return 1
	}
	return page
}

func bookIDInClause(bookIDs []int) (string, []interface{}) {
	if len(bookIDs) == 0 {
		return "0", nil
	}
	placeholders := make([]string, len(bookIDs))
	args := make([]interface{}, len(bookIDs))
	for i, id := range bookIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	return joinStrings(placeholders, ","), args
}

func joinStrings(parts []string, sep string) string {
	return strings.Join(parts, sep)
}
