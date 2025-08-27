package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidRuntime = errors.New("invaild runtime")

type Runtime int32

func (runtime *Runtime) MarshalJSON() ([]byte, error) {
	formattedJSONValue := fmt.Sprintf("%d mins", *runtime)
	quotedJSONValue := strconv.Quote(formattedJSONValue)
	return []byte(quotedJSONValue), nil
}

func (runtime *Runtime) UnmarshalJSON(JSONValue []byte) error {
	unquotedJSONValue, err := strconv.Unquote(string(JSONValue))
	if err != nil {
		return err
	}
	parts := strings.Split(string(unquotedJSONValue), " ")
	if len(parts) != 2 || parts[1] != "mins" {
		return ErrInvalidRuntime
	}

	runtimeInt, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidRuntime
	}

	*runtime = Runtime(runtimeInt)

	return nil
}
