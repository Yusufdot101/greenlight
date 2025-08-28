package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Yusufdot101/greenlight/internal/data"
	"github.com/Yusufdot101/greenlight/internal/validator"
	"golang.org/x/time/rate"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				app.serverError(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func (app *application) rateLimiter(next http.Handler) http.Handler {
	type client struct {
		limiter  rate.Limiter
		lastSeen time.Time
	}
	var (
		clients = make(map[string]*client)
		mu      sync.Mutex
	)

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

		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		mu.Lock()
		if _, exists := clients[ip]; !exists {
			clients[ip] = &client{
				limiter: *rate.NewLimiter(
					rate.Limit(app.config.limiter.requestsPerSecond), app.config.limiter.burst,
				),
			}
		}
		clients[ip].lastSeen = time.Now()
		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			app.rateLimitExceededResponse(w)
			return
		}

		mu.Unlock()
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func (app *application) authenticate(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Authorization")

		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		headParts := strings.Split(authorizationHeader, " ")
		if len(headParts) != 2 || headParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w)
			return
		}

		token := headParts[1]
		v := validator.NewValidator()

		if data.ValidateTokenPlaintext(v, token); !v.IsValid() {
			app.invalidAuthenticationTokenResponse(w)
			return
		}

		user, err := app.models.Users.GetUserForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrNoRecord):
				app.invalidAuthenticationTokenResponse(w)
			default:
				app.serverError(w, r, err)
			}
			return
		}

		r = app.contextSetUser(r, user)
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		if !user.Activated {
			app.inactiveAccountResponse(w)
			return
		}
		next.ServeHTTP(w, r)
	}

	return app.requireAuthenticatedUser(fn)
}

func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w)
			return
		}
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func (app *application) requirePermission(
	permission string, next http.HandlerFunc,
) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		permissions, err := app.models.Permissions.GellAllForUser(user.ID)
		if err != nil {
			app.serverError(w, r, err)
			return
		}
		if !permissions.Include(permission) {
			app.notPermittedResponse(w)
			return
		}

		next.ServeHTTP(w, r)
	}
	return app.requireActivatedUser(fn)
}
