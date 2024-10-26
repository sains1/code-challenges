package cmd

import "github.com/codecrafters-io/redis-starter-go/app/resp"

func HandleXRange(ctx HandleContext) (string, error) {

	streamkey := ctx.RespArr.Elements[1].(*resp.RespBulkString)
	start := ctx.RespArr.Elements[2].(*resp.RespBulkString)
	end := ctx.RespArr.Elements[3].(*resp.RespBulkString)

	ctx.Logger.Info().
		Str("stream_key", streamkey.Content).
		Str("start", start.Content).
		Str("end", end.Content).
		Msg("xrange handler")

	r, err := ctx.HostCtx.Store.GetStream(streamkey.Content, start.Content, end.Content)
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

	return resp.NewRespArray(result).AsRespString(), nil
}
