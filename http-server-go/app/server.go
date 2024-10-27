package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path"

	"github.com/codecrafters-io/http-server-starter-go/app/lib/http"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := log.With().Logger()

	filedir := flag.String("directory", "wwwroot", "Path to the files directory")
	port := *flag.Int("port", 4221, "port to listen on")

	flag.Parse()

	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		logger.Fatal().Msgf("failed to bind port %d", port)
		os.Exit(1)
	}

	logger.Info().Msgf("listening on port %d", port)

	// Root Handler
	root, _ := http.NewRouteHandler("GET", "/", func(req http.HttpRequest, res http.HttpResponse) {
		req.Logger.Info().Msg("handling root command")
		res.Status = http.Status200OK
		res.Send()
	})

	// Echo handler
	echo, _ := http.NewRouteHandler("GET", "/echo/{command}", func(req http.HttpRequest, res http.HttpResponse) {
		command, exists := req.RouteVars["command"]
		req.Logger.Info().Str("command", command).Msg("handling echo command")
		if exists {
			res.SendPlain(command)
		} else {
			res.Status = http.Status500InternalServerError // todo better error
			res.Send()
		}
	})

	// User-Agent handler
	uagent, _ := http.NewRouteHandler("GET", "/user-agent", func(req http.HttpRequest, res http.HttpResponse) {
		uagentheader, exists := req.Headers["user-agent"]
		req.Logger.Info().Str("uagent", uagentheader).Msg("handling uagent command")

		if !exists {
			res.Status = http.Status500InternalServerError // todo better error
			res.SendPlain("expected user-agent header to be sent")
		} else {
			res.SendPlain(uagentheader)
		}
	})

	// File read handler
	fileread, _ := http.NewRouteHandler("GET", "/files/{fileName}", func(req http.HttpRequest, res http.HttpResponse) {
		filename, exists := req.RouteVars["fileName"]
		req.Logger.Info().Str("filename", filename).Str("dir", *filedir).Msg("handling file command")

		if !exists {
			res.Status = http.Status500InternalServerError // todo better error
			res.Send()
			return
		}

		f, err := os.Open(path.Join(*filedir, filename))
		if err != nil {
			req.Logger.Error().Err(err).Msg("error reading file")
			if os.IsNotExist(err) {
				res.Status = http.Status404NotFound
				res.Send()
			} else {
				res.Status = http.Status500InternalServerError // todo better error
				res.Send()
			}
			return
		}
		defer f.Close()

		res.SendFileStream(f)
	})

	// File write handler
	filewrite, _ := http.NewRouteHandler("POST", "/files/{fileName}", func(req http.HttpRequest, res http.HttpResponse) {
		filename, exists := req.RouteVars["fileName"]
		req.Logger.Info().Str("filename", filename).Str("dir", *filedir).Str("body", req.Body).Msg("handling write file command")

		if !exists {
			req.Logger.Error().Msg("expeced route param didn't exist")
			res.Status = http.Status500InternalServerError // todo better error
			res.Send()
			return
		}

		file, err := os.Create(path.Join(*filedir, filename))
		if err != nil {
			req.Logger.Error().Str("filename", filename).Msg("unable to create file")
			res.Status = http.Status500InternalServerError // todo better error
			res.Send()
			return
		}
		defer file.Close()

		_, err = file.Write([]byte(req.Body))
		if err != nil {
			req.Logger.Error().Str("filename", filename).Msg("unable to write content to file")
			res.Status = http.Status500InternalServerError // todo better error
			res.Send()
			return
		}

		res.Status = http.Status201Created
		res.Send()
	})

	handlers := []http.RouteHandler{root, echo, uagent, fileread, filewrite}
	pipeline := http.NewHttpPipeline(handlers, logger)

	for {
		conn, err := l.Accept()
		if err != nil {
			logger.Fatal().Err(err).Msg("Error accepting connection")
			os.Exit(1)
		}

		logger.Debug().Msg("Accepted request")

		go pipeline.Handle(conn)
	}
}
