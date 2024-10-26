package cmd

import (
	"fmt"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

const (
	LeaderRole   = "master"
	FollowerRole = "slave"
)

func HandleInfo(ctx HandleContext) (string, error) {
	role := LeaderRole
	if ctx.HostCtx.LeaderAddr != "" {
		role = FollowerRole
	}

	info := make([]string, 0, 10)
	info = append(info, fmt.Sprintf("role:%s", role))

	if ctx.HostCtx.LeaderReplId != "" {
		info = append(info, fmt.Sprintf("master_replid:%s", ctx.HostCtx.LeaderReplId))
		info = append(info, fmt.Sprintf("master_repl_offset:%d", 0))
	}

	return resp.NewRespBulkString(strings.Join(info, "\r\n") + "\r\n").AsRespString(), nil
}
