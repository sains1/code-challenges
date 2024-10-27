package http

import (
	"bufio"
	"net"

	"github.com/rs/zerolog"
)

type HttpRequest struct {
	HttpMethod  string
	HttpVersion string
	Path        string
	RouteVars   map[string]string
	Headers     map[string]string
	Body        string
	Logger      zerolog.Logger
}

type HttpResponse struct {
	Status        string
	Body          *bufio.Reader
	ContentType   string
	ContentLength int64
	HttpVersion   string
	Request       HttpRequest
	Encoding      string

	conn net.Conn
}
