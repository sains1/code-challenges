package http

import (
	"bufio"
	"net"
	"os"
	"strconv"

	"github.com/rs/zerolog"
)

type HttpPipeline struct {
	handlers []RouteHandler
	logger   zerolog.Logger
}

func NewHttpPipeline(handlers []RouteHandler, logger zerolog.Logger) HttpPipeline {
	return HttpPipeline{
		handlers: handlers,
		logger:   logger,
	}
}

var NotFoundHandlerFunc HandlerFunction = func(req HttpRequest, res HttpResponse) {
	req.Logger.Info().Msg("404 root not found")
	res.Status = Status404NotFound
	res.Send()
}

func (p *HttpPipeline) Handle(conn net.Conn) {

	reader := bufio.NewReader(conn)

	// read request line
	rline, err := readRequestLine(reader)
	if err != nil {
		p.logger.Fatal().Err(err).Msg("failed to read request line") // todo this should just return an error code
		os.Exit(1)
	}

	// read headers
	headers, err := readHeaderLines(reader)
	if err != nil {
		p.logger.Fatal().Err(err).Msg("failed to read headers") // todo this should just return an error code
		os.Exit(1)
	}

	p.logger.Debug().Interface("headers", headers).Msg("Parsed headers")

	// read body
	t, texist := headers["content-type"]
	l, lexist := headers["content-length"]
	body := ""
	if texist && lexist {
		length, _ := strconv.Atoi(l)
		body, err = readBody(reader, length)

		if err != nil {
			p.logger.Fatal().Err(err).Msg("failed to read body") // todo this should just return an error code
			os.Exit(1)
		}
	} else {
		p.logger.Debug().Str("content-type", t).Str("content-length", l).Msg("skipping reading body as either content-type or content-length not sent")
	}

	req := HttpRequest{
		HttpMethod:  rline.HttpMethod,
		HttpVersion: rline.HttpVersion,
		Path:        rline.RequestPath,
		Body:        body,
		Headers:     headers,
		Logger:      p.logger.With().Str("path", rline.RequestPath).Str("method", rline.HttpMethod).Logger(),
	}

	res := HttpResponse{
		Status:      Status200OK,
		HttpVersion: Http1Dot1Version,
		Request:     req,
		conn:        conn,
		Encoding:    headers["accept-encoding"],
	}

	p.logger.Debug().Str("method", req.HttpMethod).Str("path", req.Path).Msgf("parsed request deets")
	var handlerFunc HandlerFunction = NotFoundHandlerFunc

	for _, h := range p.handlers {
		p.logger.Debug().Msgf("trying %s %s", h.method, h.pattern)
		if (req.HttpMethod) != h.method {
			p.logger.Debug().Msgf("no method match")
			continue
		}

		match, vars := match(req.Path, h.segments)
		if !match {
			p.logger.Debug().Msgf("no match %s %s", h.method, h.pattern)
			continue
		}

		p.logger.Debug().Msgf("found match! %s %s", h.method, h.pattern)
		handlerFunc = h.handle
		req.RouteVars = vars
		break
	}

	handlerFunc(req, res)
}
