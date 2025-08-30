package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Yusufdot101/greenlight/internal/data"
	"github.com/Yusufdot101/greenlight/internal/jsonlog"
	"github.com/Yusufdot101/greenlight/internal/mailer"
	_ "github.com/lib/pq"
)

var (
	version   string
	buildTime string
)

type config struct {
	port int
	env  string
	db   struct {
		dsn             string
		maxOpenConns    int
		maxIdleConns    int
		idleConnTimeout string
	}
	limiter struct {
		requestsPerSecond float64
		burst             int
		enabled           bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	cors struct {
		trustedOrigins []string
	}
}

type application struct {
	config config
	models *data.Models
	logger *jsonlog.Logger
	mailer *mailer.Mailer
	wg     sync.WaitGroup
}

func main() {
	var cfg config
	port := os.Getenv("PORT")
	if port == "" {
		port = "4000" // fallback for local dev
	}
	flag.IntVar(&cfg.port, "port", mustAtoi(port), "API server port")

	flag.StringVar(&cfg.env, "env", "development", "environment (development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(
		&cfg.db.idleConnTimeout, "db-idle-conn-timeout", "15m", "PostgreSQL idle connection timout",
	)

	flag.Float64Var(
		&cfg.limiter.requestsPerSecond, "limiter-rps", 2,
		"rate limiter maximum requests per second",
	)
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 5, "rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "enable rate limiter")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "3b009b986e9a42", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "5554cb8d083921", "SMTP password")
	flag.StringVar(
		&cfg.smtp.sender, "smtp-sender", "Greenlight <noreply@greenlight.ym.net>",
		"SMTP sender",
	)

	flag.Func("cors-trusted-origins", "Trusted CORS origin (space separated)",
		func(val string) error {
			cfg.cors.trustedOrigins = strings.Fields(val)
			return nil
		},
	)
	displayVersion := flag.Bool("version", false, "Dispaly version and exit")
	minLevel := flag.Int("logger-min-levl", 0, "logger minimum severity level to log")

	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		fmt.Printf("Build Time:\t%s\n", buildTime)
		os.Exit(0)
	}

	logger := jsonlog.NewLogger(os.Stdout, jsonlog.Level(*minLevel))

	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
		return
	}
	logger.PrintIfo("connection to the db established", nil)

	expvar.NewString("version").Set(version)
	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))
	expvar.Publish("database", expvar.Func(func() any {
		return db.Stats()
	}))

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.NewMailer(
			cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender,
		),
	}

	err = app.serve()
	if err != nil {
		app.logger.PrintFatal(err, nil)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	duration, err := time.ParseDuration(cfg.db.idleConnTimeout)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func mustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}
