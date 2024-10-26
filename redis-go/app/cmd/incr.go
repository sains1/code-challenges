package cmd

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/app/replication"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
	"github.com/codecrafters-io/redis-starter-go/app/store"
)

func HandleIncr(ctx HandleContext) (string, error) {
	key := ctx.RespArr.Elements[1].(*resp.RespBulkString)
	val, exists := ctx.HostCtx.Store.Get(key.Content)

	newval := 1
	if exists {
		switch v := val.(type) {
		case string:
			{
				intval, err := strconv.Atoi(v)
				if err != nil {
					return resp.NewRespError("ERR value is not an integer or out of range").AsRespString(), nil
				}

				newval = intval + 1
			}
		default:
			return "", fmt.Errorf("unexpected value type in incr %v", reflect.TypeOf(v))
		}
	}

	ctx.HostCtx.Store.Set(key.Content, strconv.Itoa(newval), store.ValueOptions{})

	// TODO - should queued commands as part of a transaction be published or _only_ after the commit in exec?
	ctx.HostCtx.PubSubManager.EventsChannel <- replication.PubSubEvent(ctx.RespArr.AsRespString())

	return resp.NewRespInteger(newval).AsRespString(), nil
}
