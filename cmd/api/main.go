package main

import (
	"context"
	"database/sql"
	"flag"
	"os"
	"sync"
	"time"

	"github.com/Yusufdot101/greenlight/internal/data"
	"github.com/Yusufdot101/greenlight/internal/jsonlog"
	"github.com/Yusufdot101/greenlight/internal/mailer"
	_ "github.com/lib/pq"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn             string
		maxOpenConns    int
		maxIdleConns    int
		connMaxIdleTime string
	}
	limiter struct {
		requstsPerSecond float64
		burst            int
		enabled          bool
	}
	stmp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
}

type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "addr", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment(development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.connMaxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max idle time")

	flag.Float64Var(
		&cfg.limiter.requstsPerSecond, "limiter-rps", 2, "Rate limiter maximum requests per second",
	)
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.StringVar(&cfg.stmp.host, "stmp-host", "sandbox.smtp.mailtrap.io", "STMP host")
	flag.IntVar(&cfg.stmp.port, "stmp-port", 25, "STMP port")
	flag.StringVar(&cfg.stmp.username, "stmp-username", "3b009b986e9a42", "STMP username")
	flag.StringVar(&cfg.stmp.password, "stmp-password", "5554cb8d083921", "STMP password")
	flag.StringVar(
		&cfg.stmp.sender, "stmp-sender", "Greenlight <no-reply@greenlight.yusufmohamed.net>",
		"STMP sender",
	)

	flag.Parse()

	logger := jsonlog.NewLogger(os.Stdout, 0)

	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}

	// defer a call to to db.Close() so that the connection pool is closed before
	// the main() function exists
	defer db.Close()
	logger.PrintInfo("database connection pool established", nil)

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.NewMailer(
			cfg.stmp.host, cfg.stmp.port, cfg.stmp.username, cfg.stmp.password, cfg.stmp.sender,
		),
	}

	if err = app.serve(); err != nil {
		logger.PrintFatal(err, nil)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	// create an empty connection pool using the dsn from the config
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	// set the max number of open (in-use + idle) connections in the pool.
	// value less than or equal to 0 will mean no limit
	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	// set the max number of idle connections in the pool.
	// value less than or equal to 0 will mean no limit
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	// use the time.ParseDuration() function to convert the idle timeout duration
	// string to time.Duration type
	duration, err := time.ParseDuration(cfg.db.connMaxIdleTime)
	if err != nil {
		return nil, err
	}
	// set the max idle timeout
	db.SetConnMaxIdleTime(duration)

	// create a context with a 5-second timeout deadline
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// using the PingContext() to establish a new connection to the database,
	// passing in the context we created. if the connection couldn't be established
	// successfully within the 5 second deadline, this will return an error
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	return db, nil
}
