package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Yusufdot101/greenlight/internal/data"
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
}

type application struct {
	config config
	logger *log.Logger
	models data.Models
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "addr", 4000, "API server port")
	flag.StringVar(
		&cfg.env, "env", "development",
		"Environment(development|staging|production)",
	)
	flag.StringVar(
		&cfg.db.dsn, "db-dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL DSN",
	)
	flag.IntVar(
		&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections",
	)
	flag.IntVar(
		&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections",
	)
	flag.StringVar(
		&cfg.db.connMaxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max idle time",
	)

	flag.Parse()

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := openDB(cfg)
	if err != nil {
		logger.Fatal(err)
	}

	// defer a call to to db.Close() so that the connection pool is close before
	// the main() function exists
	defer db.Close()
	fmt.Println("database connection established")

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
	}

	srv := http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  1 * time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	fmt.Printf("Starting %s on port %d", app.config.env, app.config.port)
	err = srv.ListenAndServe()
	log.Fatal(err)
}

func openDB(cfg config) (*sql.DB, error) {
	// create an empty connection pool using the dsn from the config
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	//set the max number of open (in-use + idle) connections in the pool.
	// value less than or equal to 0 will mean no limit
	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	//set the max number of idle connections in the pool.
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
