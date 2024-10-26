package replication

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/app/rdb"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
	"github.com/rs/zerolog"
)

type ReplicationClient struct {
	Conn           net.Conn
	Reader         *bufio.Reader
	Logger         zerolog.Logger
	inboundPort    int
	leader_repl_id string
}

func NewReplicationClient(leaderServerAddress string, inboundPort int, logger zerolog.Logger) (ReplicationClient, error) {
	conn, err := net.Dial("tcp", leaderServerAddress)

	if err != nil {
		return ReplicationClient{}, err
	}

	reader := bufio.NewReader(conn)

	return ReplicationClient{
		Conn:           conn,
		Reader:         reader,
		Logger:         logger,
		inboundPort:    inboundPort,
		leader_repl_id: "?",
	}, nil
}

func (r *ReplicationClient) SendHandshake() error {
	r.Logger.Info().Msg("Pinging leader")
	err := r.sendPing()
	if err != nil {
		return err
	}

	r.Logger.Info().Msg("Sending replconf 1")
	err = r.sendReplconf1()
	if err != nil {
		return err
	}

	r.Logger.Info().Msg("Sending replconf 2")
	err = r.sendReplconf2()
	if err != nil {
		return err
	}

	r.Logger.Info().Msg("Handshake complete")
	return nil
}

// used to synchronize the state with the leader
//
// format: PSYNC <LEADER_REPL_ID> <OFFSET>
func (r *ReplicationClient) PSync() error {
	r.Logger.Info().Msg("PSyncing with leader")

	res, err := r.send(resp.NewRespArray([]resp.RespType{
		resp.NewRespBulkString("PSYNC"),
		resp.NewRespBulkString(r.leader_repl_id),
		resp.NewRespBulkString(strconv.Itoa(-1)),
	}))

	if err != nil {
		return err
	}

	r.Logger.Info().Msgf("recieved psync res: %s\n", res) // TODO response not checked yet in this stage

	// handle rdb
	_, err = rdb.DeserializeRdb(r.Reader)
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to receive rdb file from leader")
		os.Exit(1)
	}

	r.Logger.Info().Msg("got rdb")

	return nil
}

// replica pinging the master
func (r *ReplicationClient) sendPing() error {
	res, err := r.send(resp.PingCommand())

	if err != nil {
		return err
	}

	const eres = "+PONG\r\n"
	if res != eres {
		return fmt.Errorf("unexpected response, expected %q but got %q", eres, res)
	}

	return nil
}

// replica notifying the master of the port it's listening on
//
//	format: REPLCONF listening-port <PORT>
func (r *ReplicationClient) sendReplconf1() error {
	res, err := r.send(resp.NewRespArray([]resp.RespType{
		resp.NewRespBulkString("REPLCONF"),
		resp.NewRespBulkString("listening-port"),
		resp.NewRespBulkString(strconv.Itoa(r.inboundPort)),
	}))

	if err != nil {
		return err
	}

	const eres = "+OK\r\n"
	if res != eres {
		return fmt.Errorf("unexpected response, expected %q but got %q", eres, res)
	}

	return nil
}

// replica notifying the master of its capabilities
//
//	format: REPLCONF capa psync2
func (r *ReplicationClient) sendReplconf2() error {
	res, err := r.send(resp.NewRespArray([]resp.RespType{
		resp.NewRespBulkString("REPLCONF"),
		resp.NewRespBulkString("capa"), // hardcoded capabilities for now
		resp.NewRespBulkString("psync2"),
		resp.NewRespBulkString(string(r.inboundPort)),
	}))

	if err != nil {
		return err
	}

	const eres = "+OK\r\n"
	if res != eres {
		return fmt.Errorf("unexpected response, expected %q but got %q", eres, res)
	}

	return nil
}

func (r *ReplicationClient) send(req resp.RespType) (string, error) {
	_, err := r.Conn.Write([]byte(req.AsRespString()))
	if err != nil {
		return "", err
	}

	res, err := r.Reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return res, nil
}
