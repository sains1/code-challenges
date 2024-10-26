package resp

import (
	"fmt"
	"strings"
)

type RespType interface {
	Type() string
	PrettyPrint()
	AsRespString() string
}

// Explicit implementations
var (
	_ RespType = (*RespSimpleString)(nil)
	_ RespType = (*RespError)(nil)
	_ RespType = (*RespBulkString)(nil)
	_ RespType = (*RespArray)(nil)
)

// Constants
const (
	RespSimpleStringType string = "SimpleString"
	RespErrorType        string = "Error"
	RespBulkStringType   string = "BulkString"
	RespArrayType        string = "Array"
	RespIntegerType      string = "Integer"
)

// Simple String
type RespSimpleString struct {
	Value string
}

func NewRespSimpleString(value string) *RespSimpleString {
	return &RespSimpleString{Value: value}
}

func (s RespSimpleString) Type() string {
	return RespSimpleStringType
}

func (s *RespSimpleString) PrettyPrint() {
	fmt.Printf("type: %s, value: %s\n", s.Type(), s.Value)
}

func (s *RespSimpleString) AsRespString() string {
	return fmt.Sprintf("+%s\r\n", s.Value)
}

// Error
type RespError struct {
	Message string
}

func NewRespError(message string) *RespError {
	return &RespError{Message: message}
}

func (e RespError) Type() string {
	return RespErrorType
}

func (s *RespError) PrettyPrint() {
	fmt.Printf("type: %s, value: %s\n", s.Type(), s.Message)
}

func (s *RespError) AsRespString() string {
	return fmt.Sprintf("-%s\r\n", s.Message)
}

// Bulk String
type RespBulkString struct {
	Content string
}

func NewRespBulkString(content string) *RespBulkString {
	return &RespBulkString{Content: content}
}

func (b RespBulkString) Type() string {
	return RespBulkStringType
}

func (s *RespBulkString) PrettyPrint() {
	fmt.Printf("type: %s, value: %s\n", s.Type(), s.Content)
}

func (s *RespBulkString) AsRespString() string {
	if len(s.Content) == 0 {
		return "$-1\r\n" // null bulk string
	}

	return fmt.Sprintf("$%d\r\n%s\r\n", len(s.Content), s.Content)
}

// Array
type RespArray struct {
	Elements []RespType
}

func NewRespArray(elements []RespType) *RespArray {
	return &RespArray{Elements: elements}
}

func (r *RespArray) Type() string {
	return RespArrayType
}

func (s *RespArray) PrettyPrint() {
	fmt.Printf("type: %s\n<---values", s.Type())

	for _, item := range s.Elements {
		item.PrettyPrint()
	}

	fmt.Print("<---end\n")
}

func (s *RespArray) AsRespString() string {
	// format of array:
	// *<count_in_arr> \r\n <item1> \r\n <item2> \r\n <...items...> \r\n

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("*%d\r\n", len(s.Elements)))
	for _, item := range s.Elements {
		builder.Write([]byte(item.AsRespString()))
	}

	return builder.String()
}

// Integer
type RespInteger struct {
	Value int
}

func NewRespInteger(value int) *RespInteger {
	return &RespInteger{Value: value}
}

func (s RespInteger) Type() string {
	return RespIntegerType
}

func (s *RespInteger) PrettyPrint() {
	fmt.Printf("type: %s, value: %d\n", s.Type(), s.Value)
}

func (s *RespInteger) AsRespString() string {
	return fmt.Sprintf(":%d\r\n", s.Value)
}
