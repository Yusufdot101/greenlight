package jsonlog

import (
	"encoding/json"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

// Level type to represent the level of severity
type Level int8

// constants which represent different levels of severity. we use iota as shortcut
// to make successive values, 0, lowest -> 3
const (
	LevelInfo Level = iota
	LevelError
	LevelFatal
	LevelOff
)

// String is helper function to make the Level values human-friendly
func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return ""
	}
}

// Logger is custom type. this will holds the output destination, where the log
// entries will be written to, mininum severity level that log entries will
// written for and mutex for coordinating the writes
type Logger struct {
	out      io.Writer
	minLevel Level
	mu       sync.Mutex
}

func NewLogger(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
	}
}

// PrintInfo and the ones below are helper functions for writting log entries
// at different level
func (logger *Logger) PrintInfo(message string, properties map[string]string) {
	logger.print(LevelInfo, message, properties)
}

func (logger *Logger) PrintError(err error, properties map[string]string) {
	logger.print(LevelError, err.Error(), properties)
}

func (logger *Logger) PrintFatal(err error, properties map[string]string) {
	logger.print(LevelFatal, err.Error(), properties)
	// terminate the application for fatal level
	os.Exit(1)
}

// Print is an internal function to write the log entry
func (logger *Logger) print(
	level Level, message string, properties map[string]string,
) (int, error) {
	// if the severity level of the log entry is below the mininum severity for
	//the logger, return with no further action
	if level < logger.minLevel {
		return 0, nil
	}

	// declare an anonymous struct to hold the data for the log entry
	aux := struct {
		Level      string            `json:"level"`
		Message    string            `json:"message"`
		Time       string            `json:"time"`
		Properties map[string]string `json:"properties,omitempty"`
		Trace      string            `json:"trace,omitempty"`
	}{
		Level:      level.String(),
		Message:    message,
		Properties: properties,
		Time:       time.Now().UTC().Format(time.RFC3339),
	}

	// include the stack trace at the ERROR and FATAL levels
	if level >= LevelError {
		aux.Trace = string(debug.Stack())
	}

	// declare a line variable that will hold the log entry text
	var line []byte

	// marshal the anonymous struct to JSON and store it in the line variable
	line, err := json.Marshal(aux)
	// if there is a problem set the contents of the log entry to be plain text
	// error message
	if err != nil {
		line = []byte(LevelError.String() +
			": unable to marshal log message" + err.Error(),
		)
	}

	// lock the mutex so that no two writes to the output destination can happen
	// concurrently. if we don't do this, text for two or more log entries might
	// be intermingled in the output
	logger.mu.Lock()
	defer logger.mu.Unlock()

	// write the log entry followed by a new line
	return logger.out.Write(append(line, '\n'))
}

// Write method is implemented on the logger so that is satifies the io.Writer
// interface
func (logger *Logger) Write(message []byte) {
	logger.print(LevelError, string(message), nil)
}
