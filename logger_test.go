package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogLevel(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarn, "WARN"},
		{LogLevelError, "ERROR"},
		{LogLevel(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("LogLevel(%d).String() = %s, want %s", tt.level, got, tt.expected)
		}
	}
}

func TestLoggerOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, LogLevelDebug)

	logger.Debug("node-1", "debug message")
	logger.Info("node-1", "info message")
	logger.Warn("node-1", "warn message")
	logger.Error("node-1", "error message")

	output := buf.String()

	if !strings.Contains(output, "[DEBUG]") {
		t.Error("expected DEBUG log")
	}
	if !strings.Contains(output, "[INFO]") {
		t.Error("expected INFO log")
	}
	if !strings.Contains(output, "[WARN]") {
		t.Error("expected WARN log")
	}
	if !strings.Contains(output, "[ERROR]") {
		t.Error("expected ERROR log")
	}
	if !strings.Contains(output, "[node-1]") {
		t.Error("expected node ID in log")
	}
}

func TestLoggerLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, LogLevelWarn)

	logger.Debug("", "debug message")
	logger.Info("", "info message")
	logger.Warn("", "warn message")
	logger.Error("", "error message")

	output := buf.String()

	if strings.Contains(output, "[DEBUG]") {
		t.Error("DEBUG should be filtered")
	}
	if strings.Contains(output, "[INFO]") {
		t.Error("INFO should be filtered")
	}
	if !strings.Contains(output, "[WARN]") {
		t.Error("expected WARN log")
	}
	if !strings.Contains(output, "[ERROR]") {
		t.Error("expected ERROR log")
	}
}

func TestLoggerSetLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, LogLevelError)

	logger.Info("", "should not appear")

	if strings.Contains(buf.String(), "should not appear") {
		t.Error("INFO should be filtered at ERROR level")
	}

	logger.SetLevel(LogLevelInfo)
	logger.Info("", "should appear")

	if !strings.Contains(buf.String(), "should appear") {
		t.Error("INFO should appear after SetLevel")
	}
}

func TestLoggerWithoutNodeID(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, LogLevelInfo)

	logger.Info("", "message without node")

	output := buf.String()

	// Should not have empty brackets
	if strings.Contains(output, "[]") {
		t.Error("should not have empty brackets for nodeID")
	}
	if !strings.Contains(output, "message without node") {
		t.Error("expected message in output")
	}
}

func TestLoggerFormatArgs(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, LogLevelInfo)

	logger.Info("node-1", "count: %d, name: %s", 42, "test")

	output := buf.String()

	if !strings.Contains(output, "count: 42, name: test") {
		t.Errorf("expected formatted message, got: %s", output)
	}
}
