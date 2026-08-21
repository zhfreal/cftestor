package logger

import (
	"testing"
)

func TestLoggerLevels(t *testing.T) {
	l := NewLogger(LogLevelInfo)

	if !l.matchLogLevel(LogLevelInfo) {
		t.Errorf("expected LogLevelInfo to match logger with LogLevelInfo")
	}
	if !l.matchLogLevel(LogLevelWarning) {
		t.Errorf("expected LogLevelWarning to match logger with LogLevelInfo")
	}
	if !l.matchLogLevel(LogLevelError) {
		t.Errorf("expected LogLevelError to match logger with LogLevelInfo")
	}
	if l.matchLogLevel(LogLevelDebug) {
		t.Errorf("expected LogLevelDebug NOT to match logger with LogLevelInfo")
	}

	debugLogger := NewLogger(LogLevelDebug)
	if !debugLogger.matchLogLevel(LogLevelDebug) {
		t.Errorf("expected LogLevelDebug to match debugLogger")
	}
	if !debugLogger.matchLogLevel(LogLevelInfo) {
		t.Errorf("expected LogLevelInfo to match debugLogger")
	}

	fatalLogger := NewLogger(LogLevelFatal)
	if fatalLogger.matchLogLevel(LogLevelInfo) {
		t.Errorf("expected fatalLogger NOT to match LogLevelInfo")
	}
	if !fatalLogger.matchLogLevel(LogLevelFatal) {
		t.Errorf("expected fatalLogger to match LogLevelFatal")
	}
}

func TestGetLogLevelString(t *testing.T) {
	l := NewLogger(LogLevelInfo)
	tests := []struct {
		lv       LogLevel
		expected string
	}{
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarning, "WARNING"},
		{LogLevelError, "ERROR"},
		{LogLevelFatal, "FATAL"},
	}

	for _, tt := range tests {
		got := l.getLogLevelString(tt.lv)
		if got != tt.expected {
			t.Errorf("getLogLevelString(%v) = %q, want %q", tt.lv, got, tt.expected)
		}
	}
}
