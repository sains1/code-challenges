package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/app/cmd"
	"github.com/codecrafters-io/redis-starter-go/app/replication"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
	"github.com/codecrafters-io/redis-starter-go/app/store"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ServerConfig struct {
	Port       int
	LeaderAddr string
	Debug      bool
	DbFilename string
	Dir        string
}

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := log.With().Str("component", "main").Logger()

	conf := parseArgs(logger)

	if conf.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	logger.Debug().Interface("config", conf).Msg("Parsed config")

	hostctx := cmd.HostContext{
		Store:         store.NewKvStore(logger.With().Str("component", "kvstore").Logger()),
		ConfigStore:   store.NewKvStore(logger.With().Str("component", "confstore").Logger()),
		TxQueue:       make(map[uuid.UUID][]cmd.QueuedCommand),
		LeaderAddr:    conf.LeaderAddr,
		Port:          conf.Port,
		LeaderReplId:  "",
		PubSubManager: replication.NewPubSubManager(logger.With().Str("component", "pubsubmgr").Logger()),
		Logger:        logger,
	}

	// TODO move this into rdb init?
	hostctx.ConfigStore.Set("dir", conf.Dir, store.ValueOptions{})
	hostctx.ConfigStore.Set("dbfilename", conf.DbFilename, store.ValueOptions{})

	hostctx.Store.InitialiseFromRdbFile(conf.Dir, conf.DbFilename)

	hostctx.PubSubManager.Start()

	is_leader := conf.LeaderAddr == ""

	if is_leader {
		// leader initiation steps
		logger.Info().Msg("Starting leader initiation steps...")
		hostctx.LeaderReplId = replication.GenerateReplId()
	} else {
		// follower initiation steps
		logger.Info().Msg("Starting follower initiation steps...")
		repl_client, err := replication.NewReplicationClient(conf.LeaderAddr, conf.Port, logger.With().Str("component", "replclient").Logger())
		if err != nil {
			logger.Error().Err(err).Msg("error creating connection to leader")
			os.Exit(1)
		}

		err = repl_client.SendHandshake()
		if err != nil {
			logger.Fatal().Stack().Err(err).Msg("Error sending handshake")
			os.Exit(1)
		}

		err = repl_client.PSync()
		if err != nil {
			logger.Fatal().Stack().Err(err).Msg("Error psyncing")
			os.Exit(1)
		}

		startReplicationListener(&hostctx, repl_client.Conn, repl_client.Reader)
	}

	// begin serving
	address := fmt.Sprintf("0.0.0.0:%d", conf.Port)

	l, err := net.Listen("tcp", address)
	if err != nil {
		logger.Fatal().Int("port", conf.Port).Msg("Failed to bind to port")
		os.Exit(1)
	}

	logger.Info().Str("address", address).Msg("Waiting for connection")
	var wg sync.WaitGroup

	// main request loop
	wg.Add(1)
	go func() {
		defer wg.Done()

		logger.Info().Msg("Connection loop is accepting requests...")
		for {
			conn, err := l.Accept()
			if err != nil {
				logger.Err(err).Msg("Error accepting connection")
				continue
			}

			go handleConnection(conn, &hostctx)
		}
	}()

	// Wait for the request loop goroutine to finish
	wg.Wait()
}

func startReplicationListener(hostctx *cmd.HostContext, conn net.Conn, reader io.Reader) {
	logger := log.With().Str("component", "repl_listener").Logger()
	go func() {
		lexer := resp.NewLexer(reader)
		parser := resp.NewParser(lexer)

		for {
			p := hostctx.ProcessedBytes
			c, arr, err := cmd.ParseServerCommand(*parser)
			if err != nil {
				if err == io.EOF {
					logger.Debug().Msg("EOF")
					break
				}

				logger.Err(err).Msg("error parsing the server command")
				continue // TODO better handling for EOF vs other errs
			}

			logger.Info().Int("elements", len(arr.Elements)).Str("command", c).Msg("got replication command")

			commandCtx := cmd.HandleContext{
				Conn:    conn,
				HostCtx: hostctx,
				RespArr: arr,
				Logger:  logger.With().Str("command", c).Logger(),
				ConnId:  uuid.New(),
			}

			res, err := cmd.HandleCommand(commandCtx, c)
			if err != nil {
				logger.Err(err).Msg("error handling command")
				continue
			}

			// deliberately don't send response to conn for most replication commands
			//		TODO need a better way to handle this
			if strings.ToLower(c) == "replconf" && strings.ToLower(arr.Elements[1].(*resp.RespBulkString).Content) == "getack" {
				conn.Write([]byte(res))
			}

			hostctx.AppendProcessedBytes(lexer.ByteCounter - p)
			logger.Debug().Int("processed_bytes", lexer.ByteCounter-p).Msg("Processed bytes")
		}
	}()

	logger.Info().Msg("Starting replication listener...")
}

func handleConnection(conn net.Conn, hostctx *cmd.HostContext) {
	logger := log.With().Str("component", "conn_listener").Logger()

	defer conn.Close()
	connId := uuid.New()

	lexer := resp.NewLexer(conn)
	parser := resp.NewParser(lexer)
	for {
		c, arr, err := cmd.ParseServerCommand(*parser)
		if err != nil {
			if err == io.EOF {
				logger.Debug().Msg("EOF")
				return
			}
			logger.Err(err).Msg("error parsing the server command")
			return // TODO send an error message to client
		}

		commandCtx := cmd.HandleContext{
			Conn:    conn,
			HostCtx: hostctx,
			RespArr: arr,
			Logger:  logger.With().Str("command", c).Logger(),
			ConnId:  connId,
		}

		res, err := cmd.HandleCommand(commandCtx, c)
		if err != nil {
			logger.Err(err).Msg("error handling command")
			return // TODO send an error message to client
		}

		if res != "" {
			conn.Write([]byte(res))
		}
	}
}

func parseArgs(logger zerolog.Logger) ServerConfig {
	port := 6379      // port to listen on
	leader_addr := "" // upstream addr for leader
	debug := false
	dbfilename := "dump.rdb"
	dir := "/tmp/redis-files/"

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--port":
			if i+1 < len(os.Args) {
				p, err := strconv.Atoi(os.Args[i+1])
				if err != nil {
					logger.Error().Err(err).Msg("Invalid port")
					os.Exit(1)
				}
				port = p
				i++
			} else {
				logger.Error().Msg("Missing value for --port")
				os.Exit(1)
			}
		case "--replicaof":
			if i+1 < len(os.Args) {
				replicaof := os.Args[i+1]

				parts := strings.Split(replicaof, " ")
				if len(parts) != 2 {
					logger.Error().Msg("Invalid value for replicaof expected '<host> <port>'")
					os.Exit(1)
				}

				host := parts[0]
				port, err := strconv.Atoi(parts[1])

				if err != nil {
					logger.Error().Err(err).Msg("Invalid port")
					os.Exit(1)
				}

				leader_addr = fmt.Sprintf("%s:%d", host, port)

				i++
			} else {
				logger.Error().Msg("Missing value for --replicaof")
				os.Exit(1)
			}
		case "--debug":
			debug = true
		case "--dbfilename":
			if i+1 < len(os.Args) {
				dbfilename = os.Args[i+1]
				i++
			} else {
				logger.Error().Msg("Missing value for --dbfilename")
				os.Exit(1)
			}

		case "--dir":
			if i+1 < len(os.Args) {
				dir = os.Args[i+1]
				i++
			} else {
				logger.Error().Msg("Missing value for --dir")
				os.Exit(1)
			}
		default:
			logger.Error().Str("arg", os.Args[i]).Msg("Unknown argument")
			os.Exit(1)
		}
	}

	return ServerConfig{
		Port:       port,
		LeaderAddr: leader_addr,
		Debug:      debug,
		DbFilename: dbfilename,
		Dir:        dir,
	}
}
