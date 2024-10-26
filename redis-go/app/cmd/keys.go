package cmd

import "github.com/codecrafters-io/redis-starter-go/app/resp"

func HandleKeys(ctx HandleContext) (string, error) {

	keys := ctx.HostCtx.Store.List("*")

	respBulkStrings := make([]resp.RespType, 0, len(keys))

	for _, str := range keys {
		respBulkStrings = append(respBulkStrings, resp.NewRespBulkString(str))
	}

	return resp.NewRespArray(respBulkStrings).AsRespString(), nil
}
