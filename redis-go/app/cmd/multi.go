package cmd

import "github.com/codecrafters-io/redis-starter-go/app/resp"

func HandleMulti(ctx HandleContext) (string, error) {
	ctx.HostCtx.BeginTransaction(ctx.ConnId)
	return resp.NewRespSimpleString("OK").AsRespString(), nil
}
