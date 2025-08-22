package main

import (
	"fmt"
	"net/http"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// create a differed function which will always be run in the event
		// of a panic or not
		defer func() {
			if err := recover(); err != nil {
				// if there was a panic, set a "Connection: close" header on the
				// response. this acts as a trigger to make Go's HTTP server
				// automatically close the current connectino after a response
				// has been sent
				w.Header().Set("Connection", "close")

				// the value returned by the recover() has the type interface{}
				// so we use fmt.Errorf() to normalize it into an error
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
