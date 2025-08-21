package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidRuntimeFormat = errors.New("invalid runtime format")

type Runtime int32

// MarshalJSON is how the varible will be encoded to JSON, we used custom
// logic here
func (runtime Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", runtime)

	quotedJSONValue := strconv.Quote(jsonValue)

	return []byte(quotedJSONValue), nil
}

// UnmarshalJSON is how the varible will be decoded from JSON to Golang types
// we used custom logic here
func (runtime *Runtime) UnmarshalJSON(jsonValue []byte) error {
	// we expect the json vaule will be a string in the format:
	// "<runtime> mins". we use the Unquote() to remove the quotes from the value
	unquotedJSONValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	// split the value using the " ", space, so that it becomes , [<runtime>, mins]
	parts := strings.Split(unquotedJSONValue, " ")
	if len(parts) != 2 || parts[1] != "mins" {
		return ErrInvalidRuntimeFormat
	}

	runtimeInt, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	*runtime = Runtime(runtimeInt)

	return nil
}
