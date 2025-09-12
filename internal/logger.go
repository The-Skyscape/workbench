package internal

import (
	"fmt"
	"log"
	"os"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

type Logger struct {
	level LogLevel
}

var Log = &Logger{level: INFO}

func init() {
	// Set log level from environment
	if os.Getenv("LOG_LEVEL") == "DEBUG" {
		Log.level = DEBUG
	}
}

func (l *Logger) Debug(format string, args ...any) {
	if l.level <= DEBUG {
		l.log("DEBUG", format, args...)
	}
}

func (l *Logger) Info(format string, args ...any) {
	if l.level <= INFO {
		l.log("INFO", format, args...)
	}
}

func (l *Logger) Warn(format string, args ...any) {
	if l.level <= WARN {
		l.log("WARN", format, args...)
	}
}

func (l *Logger) Error(format string, args ...any) {
	if l.level <= ERROR {
		l.log("ERROR", format, args...)
	}
}

func (l *Logger) log(level string, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("[%s] %s %s", time.Now().Format("15:04:05"), level, msg)
}
