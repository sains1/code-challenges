package cmd

import (
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

func HandleExec(ctx HandleContext) (string, error) {
	if !ctx.HostCtx.IsInTransaction(ctx.ConnId) {
		return resp.NewRespError("ERR EXEC without MULTI").AsRespString(), nil
	}

	queue := ctx.HostCtx.ConsumeTransactionQueue(ctx.ConnId)

	result := make([]string, 0, len(queue))
	for _, c := range queue {
		res, err := HandleCommand(HandleContext{
			Conn:    ctx.Conn,
			HostCtx: ctx.HostCtx,
			RespArr: c.arr,
			Logger:  ctx.Logger.With().Str("apply_from", "tx").Logger(),
		}, c.command)

		if err != nil {
			return "", fmt.Errorf("got error whilst executing transaction: %w", err)
		}

		result = append(result, res)
	}

	final := resp.NewRespArrStringFromRespStrings(result)

	ctx.Logger.Info().Msg(final)
	return final, nil
}
