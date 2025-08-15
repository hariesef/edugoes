package logger

import (
    "fmt"
    "log"
    "os"
)

// Logger is a thin wrapper around the standard logger that provides leveled logging
type Logger struct {
    *log.Logger
}

// Global logger instance
var std = &Logger{log.New(os.Stdout, "", log.LstdFlags)}

// LogLevel represents the logging level
type LogLevel int

const (
    // DebugLevel logs are typically verbose
    DebugLevel LogLevel = iota
    // InfoLevel is the default logging priority
    InfoLevel
    // WarnLevel logs are warnings
    WarnLevel
    // ErrorLevel logs are high-priority
    ErrorLevel
)

var levelNames = map[LogLevel]string{
    DebugLevel: "DEBUG",
    InfoLevel:  "INFO",
    WarnLevel:  "WARN",
    ErrorLevel: "ERROR",
}

var currentLevel = InfoLevel

// Initialize sets up the global logger level based on input string (e.g., "debug", "info", "warn", "error")
func Initialize(level string) {
    switch level {
    case "debug", "DEBUG":
        currentLevel = DebugLevel
        std.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
    case "info", "INFO", "":
        currentLevel = InfoLevel
        std.SetFlags(log.Ldate | log.Ltime)
    case "warn", "WARN", "warning", "WARNING":
        currentLevel = WarnLevel
        std.SetFlags(log.Ldate | log.Ltime)
    case "error", "ERROR":
        currentLevel = ErrorLevel
        std.SetFlags(log.Ldate | log.Ltime)
    default:
        currentLevel = InfoLevel
        std.SetFlags(log.Ldate | log.Ltime)
    }
}

func (l *Logger) log(level LogLevel, format string, v ...interface{}) {
    if level < currentLevel {
        return
    }
    prefix := fmt.Sprintf("[%s] ", levelNames[level])
    l.SetPrefix(prefix)
    l.Output(3, fmt.Sprintf(format, v...))
}

// Package-level helpers
func Debug(format string, v ...interface{}) { std.log(DebugLevel, format, v...) }
func Info(format string, v ...interface{})  { std.log(InfoLevel, format, v...) }
func Warn(format string, v ...interface{})  { std.log(WarnLevel, format, v...) }
func Error(format string, v ...interface{}) { std.log(ErrorLevel, format, v...) }
