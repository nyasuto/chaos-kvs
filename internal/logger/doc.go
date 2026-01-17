// Package logger provides a simple, thread-safe logging facility.
//
// The logger supports four levels: Debug, Info, Warn, and Error.
// Each log entry includes a timestamp, level, optional node ID, and message.
//
// # Basic Usage
//
// Using the default logger:
//
//	logger.Info("", "Application started")
//	logger.Info("node-1", "Processing request")
//	logger.Error("node-1", "Failed: %v", err)
//
// Creating a custom logger:
//
//	l := logger.New(os.Stderr, logger.LevelDebug)
//	l.Debug("node-1", "Debug message")
//
// # Log Levels
//
// Messages below the configured level are filtered:
//   - LevelDebug: all messages
//   - LevelInfo: Info, Warn, Error
//   - LevelWarn: Warn, Error
//   - LevelError: Error only
//
// # Thread Safety
//
// All logging operations are protected by a mutex and safe for concurrent use.
package logger
