package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Yusufdot101/greenlight/internal/validator"
	"github.com/julienschmidt/httprouter"
)

type envelope map[string]any

func (app *application) readIDParam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}

	return id, nil
}

func (app *application) writeJSON(
	w http.ResponseWriter, statusCode int, data envelope, headers http.Header,
) error {
	json, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}
	// Append a newline to make it easier to view in terminal applications.
	json = append(json, '\n')

	maps.Copy(w.Header(), headers)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(json)

	return nil
}

func (app *application) readJSON(
	w http.ResponseWriter, r *http.Request, dst any,
) error {
	// limit the size of the rquest to 1MB using http.MaxBytesReader
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	// initialize a new decoder and call the DisallowUnknownFields() methods on it
	// before decodig to disallow fields that cant be mapped to the destination
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	// decode the requst body to the destination
	err := decoder.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf(
				"body contains badly-formed JSON (at character%d)",
				syntaxError.Offset,
			)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		// if the type expected by the dst is not the same as the one in the
		// body, like integer instead of string
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf(
					"body contains incorrect JSON type for field %q",
					unmarshalTypeError.Field,
				)
			}
			return fmt.Errorf(
				"body contains incorrect JSON type (at charcter %d)",
				unmarshalTypeError.Offset,
			)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		// this an error caused by server and should not happen in normal
		// operations
		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		// Decode() will return an error message in the format:
		//"json: unknown field "<name>"" if there is a field that cant be mapped
		// to the target destination
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			filedName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", filedName)

		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)

		default:
			return err
		}

	}

	// call the Decode() using a pointer to an empty anonymous struct.
	// if the request body only contained one JSON s this will return an
	// io.EOF error. so anything else means we know there is additional data
	// in the request body
	err = decoder.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON s")
	}
	return nil
}

func (app *application) readString(
	qs url.Values, key string, defaultValue string,
) string {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}
	return s
}

func (app *application) readInt(
	qs url.Values, key string, defaultValue int, v *validator.Validator,
) int {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}

	return i
}

func (app *application) readCSV(
	qs url.Values, key string, defaultValue []string,
) []string {
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
