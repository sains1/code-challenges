package cmd

import (
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

func HandleXRead(ctx HandleContext) (string, error) {
	count := (len(ctx.RespArr.Elements) - 2) / 2 // -2 as arr[0] = XREAD and arr[1] = streams
	keys := make([]string, 0)
	ranges := make([]string, 0)

	for i := 2; i < 2+count; i++ {
		keys = append(keys, ctx.RespArr.Elements[i].(*resp.RespBulkString).Content)
	}

	for i := count + 2; i < len(ctx.RespArr.Elements); i++ {
		ranges = append(ranges, ctx.RespArr.Elements[i].(*resp.RespBulkString).Content)
	}

	arrs := make([]resp.RespType, 0)
	for i, key := range keys {
		rng := ranges[i]

		r, err := ctx.HostCtx.Store.XReadStream(key, rng)
		if err != nil {
			return "", err
		}
		result := make([]resp.RespType, len(r))

		for i, entry := range r {
			result[i] = resp.NewRespArray([]resp.RespType{
				resp.NewRespBulkString(entry.Seqkey),
				resp.NewRespArrFromMap(entry.Values),
			})
		}

		arrs = append(arrs, resp.NewRespArray([]resp.RespType{
			resp.NewRespBulkString(key),
			resp.NewRespArray(result),
		}))
	}

	return resp.NewRespArray(arrs).AsRespString(), nil
}
