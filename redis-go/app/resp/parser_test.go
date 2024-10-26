package resp

import (
	"strings"
	"testing"
)

func TestParser(t *testing.T) {
	tests := []struct {
		input string
		etype string
	}{
		// {"*0\r\n", RespArrayType},
		// {"*1\r\n$4\r\nping\r\n", RespArrayType},
		{"*2\r\n$4\r\necho\r\n$12\r\nhelllo world\r\n", RespArrayType},
	}

	for _, test := range tests {
		// arrange
		reader := strings.NewReader(test.input)
		lexer := NewLexer(reader)
		parser := NewParser(lexer)

		// act
		res, err := parser.Parse()

		// assert
		if err != nil {
			t.Errorf("expected no error but got %v", err)
		}

		if test.etype != res.Type() {
			t.Errorf("Expected %s but got %s", test.etype, res.Type())
		}

		///////
		if res.Type() != RespArrayType {
			t.Error() // TODO
		}

		respArray, ok := res.(*RespArray)
		if !ok {
			t.Error() // TODO
		}

		for _, item := range respArray.Elements {
			item.PrettyPrint()
		}
	}
}
