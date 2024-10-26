package cmd

import (
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

func HandleGet(ctx HandleContext) (string, error) {
	key := ctx.RespArr.Elements[1].(*resp.RespBulkString)
	val, exists := ctx.HostCtx.Store.Get(key.Content)

	if !exists {
		return resp.NullBulkString().AsRespString(), nil
	}

	var t resp.RespType

	switch v := val.(type) {
	case string:
		{
			t = resp.NewRespBulkString(v)
		}
	case int:
		{
			t = resp.NewRespInteger(v)
		}
	default:
		return "", fmt.Errorf("unexpected value type")
	}

	return t.AsRespString(), nil
}
