package http

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func (r *HttpResponse) Send() {
	r.Request.Logger.Debug().Msg("sending response")
	serializeResponse(r)
	r.conn.Close()
}

func (r *HttpResponse) SendPlain(body string) {
	r.Request.Logger.Debug().Str("body", body).Msg("sending plain body")
	r.ContentType = TextPlainContentType
	r.Body = bufio.NewReader(strings.NewReader(body))
	r.ContentLength = int64(len(body))

	r.Send()
}

func (r *HttpResponse) SendFileStream(file *os.File) error {

	r.Request.Logger.Debug().Str("filename", file.Name()).Msg("sending filestream")
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println("Error getting file info:", err)
		return err
	}

	r.ContentType = OctetStreamContentType
	r.ContentLength = fileInfo.Size()

	r.Body = bufio.NewReader(file)
	serializeResponse(r)
	r.conn.Close()

	return nil
}

type RequestLine struct {
	HttpMethod  string
	RequestPath string
	HttpVersion string
}

func readRequestLine(reader *bufio.Reader) (RequestLine, error) {
	method, err := reader.ReadString(' ')
	if err != nil {
		return RequestLine{}, err
	}

	target, err := reader.ReadString(' ')
	if err != nil {
		return RequestLine{}, err
	}

	version, err := reader.ReadString('\n')
	if err != nil {
		return RequestLine{}, err
	}

	return RequestLine{
		HttpMethod:  strings.ToLower(strings.TrimSuffix(method, " ")),
		RequestPath: strings.TrimSuffix(target, " "),
		HttpVersion: strings.TrimSuffix(version, "\r\n"),
	}, nil
}

func readHeaderLines(reader *bufio.Reader) (map[string]string, error) {

	result := make(map[string]string)

	for {
		c, err := reader.Peek(2)
		if c[0] == '\r' && c[1] == '\n' {
			// fmt.Println("crlf end of headers")
			break // end of headers
		}

		if err != nil {
			return result, err
		}

		header, err := reader.ReadString(':')
		if err != nil {
			if err == io.EOF {
				return result, fmt.Errorf("didn't expect to hit EOF reading header %w", err)
			}

			return result, err
		}

		value, err := reader.ReadString('\n')
		if err != nil {
			return result, err
		}

		header = strings.TrimSuffix(strings.ToLower(header), ":")
		value = strings.Trim(strings.ToLower(value), " \r\n")

		// fmt.Printf("got header %s\n", header)
		// fmt.Printf("got value %s\n", value)

		result[header] = value
	}

	return result, nil
}

func readBody(reader *bufio.Reader, length int) (string, error) {
	c, _ := reader.Peek(2)
	if c[0] == '\r' && c[1] == '\n' {
		reader.Discard(2)
	}

	buf := make([]byte, length)
	n, err := reader.Read(buf)

	if err != nil {
		return "", err
	}

	if n != length {
		return "", errors.New("content didn't equal expected length")
	}

	return string(buf), nil
}

/*
---------------------------------
REQUEST
---------------------------------

// Request line
GET
/user-agent
HTTP/1.1
\r\n

// Headers
Host: localhost:4221\r\n
User-Agent: foobar/1.2.3\r\n  // Read this value
Accept: *\/*\r\n
\r\n

// Request body (empty)

---------------------------------

---------------------------------
RESPONSE
---------------------------------

// Status line
HTTP/1.1 200 OK
\r\n                          // CRLF that marks the end of the status line

// Headers
Content-Type: text/plain\r\n  // Header that specifies the format of the response body
Content-Length: 3\r\n         // Header that specifies the size of the response body, in bytes
\r\n                          // CRLF that marks the end of the headers

// Response body
abc                           // The string from the request

---------------------------------
*/

var validencodings = map[string]bool{"gzip": true}

func serializeResponse(res *HttpResponse) {

	var builder strings.Builder

	// Status line
	builder.WriteString(res.HttpVersion)
	builder.WriteString(" ")
	builder.WriteString(res.Status)
	builder.WriteString("\r\n")

	// Headers
	if res.ContentType != "" {
		writeHeader(&builder, HeaderContentType, res.ContentType)
	}

	chosenencoding := ""
	if res.Encoding != "" {
		encodings := strings.Split(res.Encoding, ", ")
		res.Request.Logger.Debug().Interface("encodings", encodings).Msg("got encodings")

		for _, encoding := range encodings {
			if _, exists := validencodings[encoding]; exists {
				chosenencoding = encoding
				writeHeader(&builder, HeaderContentEncoding, chosenencoding)
				res.Request.Logger.Debug().Interface("chosen", chosenencoding).Msg("using encoding")
				break
			}
		}
	}

	switch chosenencoding {
	case "gzip":
		{
			// TODO seems a bit hacky - lose all ability to stream, look into transfer-encoding?
			body, err := gzipReaderToString(res.Body)
			if err != nil {
				os.Exit(1) // todo handle
			}

			res.ContentLength = int64(len(body))
			res.Body = bufio.NewReader(strings.NewReader(body))

			unzipped, err := decompressGzip([]byte(body))
			if err != nil {
				os.Exit(1) // todo handle
			}

			res.Request.Logger.Info().Str("zipped", body).Int("length", len(body)).Str("unzipped", unzipped).Msg("set output to gzipped reader")
		}
	}

	if res.ContentLength > 0 {
		writeHeader(&builder, HeaderContentLength, strconv.Itoa(int(res.ContentLength)))
	}

	builder.WriteString("\r\n")

	res.conn.Write([]byte(builder.String()))

	// Response Body
	if res.Body == nil {
		return
	}
	buf := make([]byte, 4096) // 4KB buffer
	for {
		n, err := res.Body.Read(buf)
		if err != nil {
			break
		}

		res.conn.Write(buf[:n])
	}
}

func writeHeader(builder *strings.Builder, key string, val string) {
	builder.WriteString(key)
	builder.WriteString(": ")
	builder.WriteString(val)
	builder.WriteString("\r\n")
}

func gzipReaderToString(reader *bufio.Reader) (string, error) {
	// Create a buffer to store the compressed data
	var compressedBuffer bytes.Buffer

	// Create a gzip writer around the buffer
	gzipWriter := gzip.NewWriter(&compressedBuffer)
	defer gzipWriter.Close()

	// Copy the contents of the reader to the gzip writer
	if _, err := io.Copy(gzipWriter, reader); err != nil {
		return "", fmt.Errorf("failed to compress data: %w", err)
	}

	// Close the gzip writer to flush all data
	if err := gzipWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Convert compressed bytes to a string
	return compressedBuffer.String(), nil
}

func decompressGzip(data []byte) (string, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var out bytes.Buffer
	if _, err := io.Copy(&out, reader); err != nil {
		return "", err
	}

	return out.String(), nil
}
