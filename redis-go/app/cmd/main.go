package cmd

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/app/replication"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
	"github.com/codecrafters-io/redis-starter-go/app/store"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type HandleContext struct {
	Conn    net.Conn
	ConnId  uuid.UUID
	HostCtx *HostContext
	RespArr resp.RespArray
	Logger  zerolog.Logger
}

type HostContext struct {
	Store          *store.KvStore
	ConfigStore    *store.KvStore
	LeaderAddr     string
	Port           int
	LeaderReplId   string
	PubSubManager  replication.PubSubManager
	Logger         zerolog.Logger
	ProcessedBytes int
	mu             sync.Mutex
	TxQueue        map[uuid.UUID][]QueuedCommand
}

type QueuedCommand struct {
	command string
	arr     resp.RespArray
}

func (h *HostContext) AppendProcessedBytes(count int) {
	h.mu.Lock()
	h.ProcessedBytes += count
	h.mu.Unlock()
}

func (h *HostContext) IsInTransaction(connid uuid.UUID) bool {
	_, exists := h.TxQueue[connid]
	return exists
}

func (h *HostContext) BeginTransaction(connid uuid.UUID) {
	h.mu.Lock()
	h.TxQueue[connid] = make([]QueuedCommand, 0)
	h.mu.Unlock()
}

func (h *HostContext) ConsumeTransactionQueue(connid uuid.UUID) []QueuedCommand {
	h.mu.Lock()
	defer h.mu.Unlock()

	queue := h.TxQueue[connid]
	delete(h.TxQueue, connid)
	return queue
}

var (
	ErrNoExistingTransaction = errors.New("no existing transaction")
)

func (h *HostContext) QueueCommand(connid uuid.UUID, c string, arr resp.RespArray) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	queue, exists := h.TxQueue[connid]
	if !exists {
		return ErrNoExistingTransaction
	}

	h.TxQueue[connid] = append(queue, QueuedCommand{command: c, arr: arr})

	return nil
}

func ParseServerCommand(p resp.Parser) (string, resp.RespArray, error) {
	c, err := p.Parse()

	if err != nil {
		if err == io.EOF {
			return "", resp.RespArray{}, err
		}
		return "", resp.RespArray{}, fmt.Errorf("error couldnt parse: %v", err)
	}

	if c.Type() != resp.RespArrayType {
		return "", resp.RespArray{}, errors.New("error expected parser output to be a resparray")
	}

	respArray, ok := c.(*resp.RespArray)
	if !ok || len(respArray.Elements) == 0 {
		return "", resp.RespArray{}, errors.New("error expected to parse parser output as a resparray")
	}

	command := respArray.Elements[0]
	if command.Type() != resp.RespBulkStringType {
		return "", resp.RespArray{}, errors.New("error expected bulk string as first element of resparray")
	}

	respCommand, ok := command.(*resp.RespBulkString)
	if !ok {
		return "", resp.RespArray{}, errors.New("error expected to parse first element of resparray as a RespBulkString")
	}

	return respCommand.Content, *respArray, nil
}

func HandleCommand(ctx HandleContext, content string) (string, error) {
	content = strings.ToLower(content)
	ctx.Logger.Info().Msgf("handling %s", content)

	if ctx.HostCtx.IsInTransaction(ctx.ConnId) && content != "exec" && content != "discard" {
		ctx.HostCtx.QueueCommand(ctx.ConnId, content, ctx.RespArr)
		return resp.NewRespSimpleString("QUEUED").AsRespString(), nil
	}

	switch content {
	case "config":
		return HandleConfig(ctx)
	case "discard":
		return HandleDiscard(ctx)
	case "echo":
		return HandleEcho(ctx)
	case "exec":
		return HandleExec(ctx)
	case "get":
		return HandleGet(ctx)
	case "incr":
		return HandleIncr(ctx)
	case "info":
		return HandleInfo(ctx)
	case "keys":
		return HandleKeys(ctx)
	case "multi":
		return HandleMulti(ctx)
	case "ping":
		return HandlePing(ctx)
	case "psync":
		HandlePSync(ctx)
		return "", nil // writes several responses direct to the conn (refactor?)
	case "replconf":
		return HandleReplconf(ctx)
	case "set":
		return HandleSet(ctx)
	case "type":
		return HandleType(ctx)
	case "wait":
		return HandleWait(ctx)
	case "xadd":
		return HandleXAdd(ctx)
	case "xrange":
		return HandleXRange(ctx)
	case "xread":
		return HandleXRead(ctx)
	default:
		ctx.Logger.Error().Msgf("unexpected command %s", content)
		panic(1)
	}
}
