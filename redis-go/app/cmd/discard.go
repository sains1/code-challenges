package cmd

import "github.com/codecrafters-io/redis-starter-go/app/resp"

func HandleDiscard(ctx HandleContext) (string, error) {

	if !ctx.HostCtx.IsInTransaction(ctx.ConnId) {
		return resp.NewRespError("ERR DISCARD without MULTI").AsRespString(), nil
	}

	_ = ctx.HostCtx.ConsumeTransactionQueue(ctx.ConnId)

	return resp.OkResponse().AsRespString(), nil
}
