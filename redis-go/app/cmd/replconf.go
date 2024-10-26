package cmd

import (
	"errors"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

func HandleReplconf(ctx HandleContext) (string, error) {
	command := ctx.RespArr.Elements[1].(*resp.RespBulkString).Content

	switch strings.ToLower(command) {
	case "listening-port":
		return resp.OkResponse().AsRespString(), nil
	case "capa":
		return resp.OkResponse().AsRespString(), nil
	case "getack":
		res := resp.AckResponse(ctx.HostCtx.ProcessedBytes).AsRespString()
		ctx.Logger.Info().Msgf("[replconf] got an ACK request, responding with: %v", res)
		return res, nil
	default:
		return "", errors.New("unexpected replconf command")
	}
}
