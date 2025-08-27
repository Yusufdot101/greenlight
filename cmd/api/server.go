package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {
	srv := http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  1 * time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	shutdownErr := make(chan error)
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit
		app.logger.PrintIfo(
			"server shutting down", map[string]string{
				"signal": s.String(),
			},
		)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownErr <- err
		}

		app.logger.PrintIfo("finishing background tasks", nil)
		app.wg.Wait()
		shutdownErr <- nil
	}()

	app.logger.PrintIfo(
		"starting server", map[string]string{
			"env":  app.config.env,
			"addr": fmt.Sprintf(":%d", app.config.port),
		},
	)
	err := srv.ListenAndServe()
	if err != http.ErrServerClosed {
		return err
	}

	err = <-shutdownErr
	if err != nil {
		return err
	}

	app.logger.PrintIfo("server stopped", map[string]string{"addr": srv.Addr})
	return nil
}
