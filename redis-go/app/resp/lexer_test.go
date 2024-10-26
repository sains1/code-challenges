package resp

import (
	"strings"
	"testing"
)

func TestNextToken(t *testing.T) {

	tests := []struct {
		input  string
		etype  TokenType
		evalue string
	}{
		{"+OK\r\n", TokenSimpleString, "OK"},
		{"-Error\r\n", TokenError, "Error"},
		{"$3\r\nfoo\r\n", TokenBulkString, "3"},
		{":123\r\n", TokenInteger, "123"},
		{"*2\r\n", TokenArray, "2"},
		{"_\r\n", TokenNull, ""},
		{"#t\r\n", TokenBool, "t"},
		{"#f\r\n", TokenBool, "f"},
	}

	for _, test := range tests {
		// arrange
		reader := strings.NewReader(test.input)
		lexer := NewLexer(reader)

		// act
		result, err := lexer.NextToken()

		// assert
		if err != nil {
			t.Error(err)
		}

		if result.Type != test.etype {
			t.Errorf("Expected next token to be %s but got %s", TokenSimpleString, result.Type)
		}

		if result.Value != test.evalue {
			t.Errorf("Expected next token value to be %s but got %s", test.evalue, result.Value)
		}
	}
}

// RESP data type	version		Category	First byte
// ---------------------------------------------------
// Simple strings	RESP2		Simple		+
// Simple Errors	RESP2		Simple		-
// Integers			RESP2		Simple		:
// Bulk strings		RESP2		Aggregate	$
// Arrays			RESP2		Aggregate	*
// Nulls			RESP3		Simple		_
// Booleans			RESP3		Simple		#
// Doubles			RESP3		Simple		,
// Big numbers		RESP3		Simple		(
// Bulk errors		RESP3		Aggregate	!
// Verbatim strings	RESP3		Aggregate	=
// Maps				RESP3		Aggregate	%
// Attributes		RESP3		Aggregate	`
// Sets				RESP3		Aggregate	~
// Pushes			RESP3		Aggregate	>

// simple strings - https://redis.io/docs/latest/develop/reference/protocol-spec/#simple-strings
// +OK\r\n

// bulk strings - https://redis.io/docs/latest/develop/reference/protocol-spec/#bulkstrings
// $-1\r\n		-> null bulk string
// $0\r\n\r\n	-> empty bulk string

// simple error - https://redis.io/docs/latest/develop/reference/protocol-spec/#simple-errors
// -Error message\r\n
// +hello world\r\n

// bulk error - https://redis.io/docs/latest/develop/reference/protocol-spec/#bulk-errors
// !21\r\nSYNTAX invalid syntax\r\n

// arrays - https://redis.io/docs/latest/develop/reference/protocol-spec/#arrays
// *0\r\n 										-> empty array
// *1\r\n$4\r\nping\r\n							-> array of 1 bulk strings (ping)
// *2\r\n$4\r\necho\r\n$11\r\nhello world\r\n	-> array of 2 bulk strings (echo, hello world)
// *2\r\n$3\r\nget\r\n$3\r\nkey\r\n				-> array of 2 bulk strings (get, key)

// nulls - https://redis.io/docs/latest/develop/reference/protocol-spec/#nulls
// _\r\n

// bools -  https://redis.io/docs/latest/develop/reference/protocol-spec/#booleans
// #t\r\n
// #f\r\n
