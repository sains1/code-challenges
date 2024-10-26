package cmd

func HandlePing(ctx HandleContext) (string, error) {
	return "+PONG\r\n", nil
}
