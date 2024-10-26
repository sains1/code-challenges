package cmd

import (
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/app/resp"
	"github.com/codecrafters-io/redis-starter-go/app/store"
)

func HandleType(ctx HandleContext) (string, error) {
	key := ctx.RespArr.Elements[1].(*resp.RespBulkString)
	val, exists := ctx.HostCtx.Store.Get(key.Content)

	if !exists {
		return resp.NewRespSimpleString("none").AsRespString(), nil
	}

	switch val.(type) {
	case string:
		{
			return resp.NewRespSimpleString("string").AsRespString(), nil
		}
	case store.Stream:
		{
			return resp.NewRespSimpleString("stream").AsRespString(), nil
		}
	default:
		return "", fmt.Errorf("unexpected value type")
	}
}
