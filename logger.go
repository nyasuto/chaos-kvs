package main

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// LogLevel はログレベルを表す
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger はスレッドセーフなロガー
type Logger struct {
	mu       sync.Mutex
	out      io.Writer
	minLevel LogLevel
}

// DefaultLogger はデフォルトのロガー
var DefaultLogger = NewLogger(os.Stdout, LogLevelInfo)

// NewLogger は新しいロガーを作成する
func NewLogger(out io.Writer, minLevel LogLevel) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
	}
}

// SetLevel はログレベルを設定する
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.minLevel = level
}

// log は指定されたレベルでログを出力する
func (l *Logger) log(level LogLevel, nodeID string, format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.minLevel {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	msg := fmt.Sprintf(format, args...)

	if nodeID != "" {
		_, _ = fmt.Fprintf(l.out, "[%s] [%s] [%s] %s\n", timestamp, level, nodeID, msg)
	} else {
		_, _ = fmt.Fprintf(l.out, "[%s] [%s] %s\n", timestamp, level, msg)
	}
}

// Debug はデバッグログを出力する
func (l *Logger) Debug(nodeID string, format string, args ...any) {
	l.log(LogLevelDebug, nodeID, format, args...)
}

// Info は情報ログを出力する
func (l *Logger) Info(nodeID string, format string, args ...any) {
	l.log(LogLevelInfo, nodeID, format, args...)
}

// Warn は警告ログを出力する
func (l *Logger) Warn(nodeID string, format string, args ...any) {
	l.log(LogLevelWarn, nodeID, format, args...)
}

// Error はエラーログを出力する
func (l *Logger) Error(nodeID string, format string, args ...any) {
	l.log(LogLevelError, nodeID, format, args...)
}

// グローバル関数（デフォルトロガーを使用）

// LogDebug はデバッグログを出力する
func LogDebug(nodeID string, format string, args ...any) {
	DefaultLogger.Debug(nodeID, format, args...)
}

// LogInfo は情報ログを出力する
func LogInfo(nodeID string, format string, args ...any) {
	DefaultLogger.Info(nodeID, format, args...)
}

// LogWarn は警告ログを出力する
func LogWarn(nodeID string, format string, args ...any) {
	DefaultLogger.Warn(nodeID, format, args...)
}

// LogError はエラーログを出力する
func LogError(nodeID string, format string, args ...any) {
	DefaultLogger.Error(nodeID, format, args...)
}
