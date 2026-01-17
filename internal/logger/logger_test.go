package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogLevel(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("Level(%d).String() = %s, want %s", tt.level, got, tt.expected)
		}
	}
}

func TestLoggerOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	l := New(buf, LevelDebug)

	l.Debug("node-1", "debug message")
	l.Info("node-1", "info message")
	l.Warn("node-1", "warn message")
	l.Error("node-1", "error message")

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
	l := New(buf, LevelWarn)

	l.Debug("", "debug message")
	l.Info("", "info message")
	l.Warn("", "warn message")
	l.Error("", "error message")

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
	l := New(buf, LevelError)

	l.Info("", "should not appear")

	if strings.Contains(buf.String(), "should not appear") {
		t.Error("INFO should be filtered at ERROR level")
	}

	l.SetLevel(LevelInfo)
	l.Info("", "should appear")

	if !strings.Contains(buf.String(), "should appear") {
		t.Error("INFO should appear after SetLevel")
	}
}

func TestLoggerWithoutNodeID(t *testing.T) {
	buf := &bytes.Buffer{}
	l := New(buf, LevelInfo)

	l.Info("", "message without node")

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
	l := New(buf, LevelInfo)

	l.Info("node-1", "count: %d, name: %s", 42, "test")

	output := buf.String()

	if !strings.Contains(output, "count: 42, name: test") {
		t.Errorf("expected formatted message, got: %s", output)
	}
}
