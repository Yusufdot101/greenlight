package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Yusufdot101/greenlight/internal/validator"
	"github.com/julienschmidt/httprouter"
)

type envelope map[string]any

func (app *application) writeJSON(w http.ResponseWriter, statusCode int, message envelope) error {
	JSON, err := json.MarshalIndent(message, "", "\t")
	if err != nil {
		return err
	}

	JSON = append(JSON, '\n')

	w.WriteHeader(statusCode)
	_, err = w.Write(JSON)
	if err != nil {
		return err
	}

	return nil
}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	const maxBytes = 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(dst)
	if err != nil {
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError
		var invaldUnmarshalErr *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxErr):
			return fmt.Errorf("body contains badly formed JSON at character: %d", syntaxErr.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return fmt.Errorf("body contains badly formed JSON")

		case errors.As(err, &unmarshalTypeErr):
			if unmarshalTypeErr.Field != "" {
				return fmt.Errorf(
					"body contains invaild type for field: %s", unmarshalTypeErr.Field,
				)
			}
			return fmt.Errorf("body contains type error at character: %d", unmarshalTypeErr.Offset)

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key: %s", fieldName)

		case err.Error() == "http: request body too large":
			return fmt.Errorf("body size cannot exceed %d bytes", maxBytes)

		case errors.Is(err, io.EOF):
			return fmt.Errorf("body cannot be empty")

		case errors.As(err, &invaldUnmarshalErr):
			panic(err.Error())

		default:
			return err
		}
	}

	err = decoder.Decode(&struct{}{})
	if err != io.EOF {
		return fmt.Errorf("body must contain only one JSON value")
	}

	return nil
}

func (app *application) readParamID(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil {
		return -1, nil
	}

	return id, nil
}

func (app *application) readString(qs url.Values, key, defaultValue string) string {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}

	return strings.ToLower(s)
}

func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}

	id, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be integer")
		return defaultValue
	}
	return id
}

func (app *application) readCSV(qs url.Values, key string, defaultValue []string) []string {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}

	return strings.Split(s, ",")
}

func (app *application) background(fn func()) {
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		defer func() {
			if err := recover(); err != nil {
				app.logger.PrintError(fmt.Errorf("%s", err), nil)
			}
		}()
		fn()
	}()
}
