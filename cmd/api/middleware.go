package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
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

func (app *application) rateLimit(next http.Handler) http.Handler {

	// a struct that will hold client info, last seen and rate limiter info so
	// that each ip has its own rate limiter instead of global one
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// will run every minute and deletes clients which were last seen more than
	// three minutes ago. this prevents the map growing indefinitely and use
	// lots of resources
	go func() {
		for {
			time.Sleep(time.Minute)

			mu.Lock()

			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}

			mu.Unlock()
		}
	}()

	fn := func(w http.ResponseWriter, r *http.Request) {
		if !app.config.limiter.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// get the ip address of the client
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		// lock to prevent this code running concurrently
		mu.Lock()
		if _, found := clients[ip]; !found {
			clients[ip] = &client{limiter: rate.NewLimiter(2, 4)}
		}

		clients[ip].lastSeen = time.Now()

		// if not permitted, ie rate limit exceeded call helper method,
		// rateLimitExceededResponse(), which will send 429 status code, too
		// many requests
		if !clients[ip].limiter.Allow() {
			// unlock the mutex
			mu.Unlock()
			app.rateLimitExceededResponse(w, r)
			return
		}

		// important, unlock the mutex
		mu.Unlock()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
