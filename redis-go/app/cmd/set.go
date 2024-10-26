package cmd

import (
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/app/replication"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
	"github.com/codecrafters-io/redis-starter-go/app/store"
)

func HandleSet(ctx HandleContext) (string, error) {
	key := ctx.RespArr.Elements[1].(*resp.RespBulkString)
	val := ctx.RespArr.Elements[2].(*resp.RespBulkString)

	ctx.Logger.Info().
		Str("key", key.Content).
		Str("value", val.Content).
		Str("type", ctx.RespArr.Elements[2].Type()).
		Msg("setting key")

	options := store.ValueOptions{}
	if len(ctx.RespArr.Elements) > 3 { // 1. set, 2. key, 3. val, [...options]
		arg1 := ctx.RespArr.Elements[3].(*resp.RespBulkString)

		if arg1.Content == "px" {
			arg1Val := ctx.RespArr.Elements[4].(*resp.RespBulkString) // TODO is this always a bulk string even tho its an int?
			expiryMs, err := strconv.Atoi(arg1Val.Content)

			if err != nil {
				ctx.Logger.Error().Msgf("Expected expiry to be an int but got %s", arg1Val.Content)
				return "", err
			}
			options.Expiry = uint64(expiryMs)
		}

	}

	err := ctx.HostCtx.Store.Set(key.Content, val.Content, options)

	ctx.HostCtx.PubSubManager.EventsChannel <- replication.PubSubEvent(ctx.RespArr.AsRespString())

	if err != nil {
		ctx.Logger.Error().Msgf("Error setting key %s with value %s: %v", key.Content, val.Content, err)
		return "", err
	}

	return resp.NewRespSimpleString("OK").AsRespString(), nil
}
