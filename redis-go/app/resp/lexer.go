package resp

import (
	"bufio"
	"fmt"
	"io"
)

type TokenType string

const (
	TokenSimpleString TokenType = "+"
	TokenError        TokenType = "-"
	TokenBulkString   TokenType = "$"
	TokenArray        TokenType = "*"
	TokenInteger      TokenType = ":"
	TokenNull         TokenType = "_"
	TokenBool         TokenType = "#"
)

type Token struct {
	Type  TokenType
	Value string
}

type Lexer struct {
	reader      *bufio.Reader
	ByteCounter int
}

func NewLexer(r io.Reader) *Lexer {
	return &Lexer{reader: bufio.NewReader(r)}
}

func (l *Lexer) NextToken() (*Token, error) {

	next, err := l.reader.ReadByte()
	l.ByteCounter++

	if err != nil {
		return nil, err
	}

	switch TokenType(next) {
	case TokenSimpleString:
		return &Token{Type: TokenSimpleString, Value: l.readLine()}, nil
	case TokenError:
		return &Token{Type: TokenError, Value: l.readLine()}, nil
	case TokenBulkString:
		return &Token{Type: TokenBulkString, Value: l.readLine()}, nil
	case TokenArray:
		return &Token{Type: TokenArray, Value: l.readLine()}, nil
	case TokenInteger:
		return &Token{Type: TokenInteger, Value: l.readLine()}, nil
	case TokenNull:
		return &Token{Type: TokenNull, Value: ""}, nil
	case TokenBool:
		return &Token{Type: TokenBool, Value: l.readLine()}, nil
	default:
		return nil, fmt.Errorf("unexpected character: %v", next)

	}
}

func (l *Lexer) readLine() string {
	line, _ := l.reader.ReadString('\n')
	l.ByteCounter += len(line)
	return line[:len(line)-2] // remove \r\n
}

func (l *Lexer) ConsumeCrlf() error {
	peekedBytes, err := l.reader.Peek(2)
	if err != nil {
		return err
	}

	if peekedBytes[0] == '\r' && peekedBytes[1] == '\n' {
		_, err = l.reader.Discard(2)
		l.ByteCounter += 2
		if err != nil {
			return err
		}
	} else {
		return io.ErrUnexpectedEOF
	}

	return nil
}

func (l *Lexer) ConsumeBytes(count int) (string, error) {
	buf := make([]byte, count)

	n, err := l.reader.Read(buf)

	if err != nil {
		return "", fmt.Errorf("failed to consume bytes %v", err)
	}

	if n != count {
		return "", fmt.Errorf("expected to read %d bytes but got %d", count, n)
	}

	l.ByteCounter += count

	return string(buf), nil
}
