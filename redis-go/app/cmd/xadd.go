package cmd

import (
	"github.com/codecrafters-io/redis-starter-go/app/resp"
	"github.com/codecrafters-io/redis-starter-go/app/store"
)

// redis-cli XADD stream_key 0-1 foo bar
func HandleXAdd(ctx HandleContext) (string, error) {
	streamkey := ctx.RespArr.Elements[1].(*resp.RespBulkString)
	seqkey := ctx.RespArr.Elements[2].(*resp.RespBulkString)
	key := ctx.RespArr.Elements[3].(*resp.RespBulkString)
	value := ctx.RespArr.Elements[4].(*resp.RespBulkString)

	ctx.Logger.Info().
		Str("key", key.Content).Str("value", value.Content).
		Str("stream_key", streamkey.Content).
		Str("seq_key", seqkey.Content).
		Msg("xadd handler")

	skey, err := ctx.HostCtx.Store.SetStream(streamkey.Content, seqkey.Content, key.Content, value.Content, store.ValueOptions{})

	if err != nil {
		return resp.NewRespError(err.Error()).AsRespString(), nil
	}

	return resp.NewRespBulkString(skey).AsRespString(), nil
}
