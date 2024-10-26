package cmd

import (
	"fmt"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

func HandleConfig(ctx HandleContext) (string, error) {
	op := ctx.RespArr.Elements[1].(*resp.RespBulkString).Content

	if strings.ToLower(op) == "get" {
		key := ctx.RespArr.Elements[2].(*resp.RespBulkString).Content
		val, exists := ctx.HostCtx.ConfigStore.Get(key)

		svalue, ok := val.(string)
		if !ok {
			return "", fmt.Errorf("expected config to be a string")
		}

		if exists {
			return resp.NewRespArray([]resp.RespType{
				resp.NewRespBulkString(key),
				resp.NewRespBulkString(svalue),
			}).AsRespString(), nil
		}

		return "", fmt.Errorf("error key not found: %v", key)

	} else {
		return "", fmt.Errorf("error unknown operation: %v", op)
	}
}
