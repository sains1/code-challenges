package resp

import (
	"fmt"
	"strconv"
	"strings"
)

func NullBulkString() *RespBulkString {
	return NewRespBulkString("")
}

func OkResponse() *RespSimpleString {
	return NewRespSimpleString("OK")
}

func PSyncResponse(replid string, offset int) *RespSimpleString {
	return NewRespSimpleString(fmt.Sprintf("FULLRESYNC %s %v", replid, offset))
}

func PingCommand() *RespArray {
	return NewRespArray([]RespType{
		NewRespBulkString("PING"),
	})
}

func AckResponse(offset int) *RespArray {
	return NewRespArray([]RespType{
		NewRespBulkString("REPLCONF"),
		NewRespBulkString("ACK"),
		NewRespBulkString(strconv.Itoa(offset)),
	})
}

// TODO hack to workarond needing to parse the items as resp types before converting to array
func NewRespArrStringFromRespStrings(items []string) string {
	// format of array:
	// *<count_in_arr> \r\n <item1> \r\n <item2> \r\n <...items...> \r\n

	fmt.Printf("!! length: %d", len(items))

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("*%d\r\n", len(items)))
	for _, item := range items {
		builder.Write([]byte(item))
	}

	return builder.String()
}

func NewRespArrFromMap(items map[string]interface{}) *RespArray {
	inner := make([]RespType, 0)

	for k, v := range items {

		inner = append(inner, NewRespBulkString(k))
		inner = append(inner, NewRespBulkString(v.(string)))
	}

	return NewRespArray(inner)
}
