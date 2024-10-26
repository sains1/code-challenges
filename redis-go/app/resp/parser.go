package resp

import (
	"fmt"
	"io"
	"strconv"
)

type Parser struct {
	lexer *Lexer
}

func NewParser(l *Lexer) *Parser {
	return &Parser{lexer: l}
}

func (p *Parser) Parse() (RespType, error) {
	token, err := p.lexer.NextToken()

	if err != nil {
		if err == io.EOF {
			return nil, err
		}

		return nil, fmt.Errorf("unable to parse next token %v", err)
	}

	switch token.Type {
	case TokenSimpleString:
		return NewRespSimpleString(token.Value), nil
	case TokenBulkString:
		return p.parseBulkString(*token)
	case TokenArray:
		return p.parseArray(*token)
	default:
		return nil, fmt.Errorf("unexpected type") // TODO
	}

}

func (p *Parser) parseBulkString(token Token) (RespType, error) {

	count, err := strconv.Atoi(token.Value)

	if err != nil {
		return nil, fmt.Errorf("invalid number of elements for resp array %v", err)
	}

	str, err := p.lexer.ConsumeBytes(count)

	if err != nil {
		return nil, fmt.Errorf("failed to read bulk string %v", err)
	}

	p.lexer.ConsumeCrlf()

	return NewRespBulkString(str), nil

}

func (p *Parser) parseArray(token Token) (RespType, error) {
	count, err := strconv.Atoi(token.Value)

	if err != nil {
		return nil, fmt.Errorf("invalid number of elements for resp array %v", err)
	}

	elements := make([]RespType, 0, count)

	for i := 0; i < count; i++ {
		element, err := p.Parse()
		if err != nil {
			return nil, err
		}

		elements = append(elements, element)
	}

	return NewRespArray(elements), nil
}
