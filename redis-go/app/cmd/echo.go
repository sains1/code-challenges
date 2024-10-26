package cmd

import "fmt"

func HandleEcho(ctx HandleContext) (string, error) {
	if len(ctx.RespArr.Elements) != 2 {
		return "", fmt.Errorf("expected echo to pass exactly 2 args: 'echo' and 'value' but got %v", len(ctx.RespArr.Elements))
	}

	echoValue := ctx.RespArr.Elements[1]

	return echoValue.AsRespString(), nil
}
