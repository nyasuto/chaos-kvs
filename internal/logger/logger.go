package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level はログレベルを表す
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger はスレッドセーフなロガー
type Logger struct {
	mu       sync.Mutex
	out      io.Writer
	minLevel Level
}

// Default はデフォルトのロガー
var Default = New(os.Stdout, LevelInfo)

// New は新しいロガーを作成する
func New(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
	}
}

// SetLevel はログレベルを設定する
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.minLevel = level
}

// log は指定されたレベルでログを出力する
func (l *Logger) log(level Level, nodeID string, format string, args ...any) {
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
	l.log(LevelDebug, nodeID, format, args...)
}

// Info は情報ログを出力する
func (l *Logger) Info(nodeID string, format string, args ...any) {
	l.log(LevelInfo, nodeID, format, args...)
}

// Warn は警告ログを出力する
func (l *Logger) Warn(nodeID string, format string, args ...any) {
	l.log(LevelWarn, nodeID, format, args...)
}

// Error はエラーログを出力する
func (l *Logger) Error(nodeID string, format string, args ...any) {
	l.log(LevelError, nodeID, format, args...)
}

// グローバル関数（デフォルトロガーを使用）

// Debug はデバッグログを出力する
func Debug(nodeID string, format string, args ...any) {
	Default.Debug(nodeID, format, args...)
}

// Info は情報ログを出力する
func Info(nodeID string, format string, args ...any) {
	Default.Info(nodeID, format, args...)
}

// Warn は警告ログを出力する
func Warn(nodeID string, format string, args ...any) {
	Default.Warn(nodeID, format, args...)
}

// Error はエラーログを出力する
func Error(nodeID string, format string, args ...any) {
	Default.Error(nodeID, format, args...)
}
