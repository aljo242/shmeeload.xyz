package log

import (
	"errors"
	"testing"
)

// TestLoggingDoesNotPanic exercises Setup and each level with the various
// key/value types handled by emit (string, int, bool, error), ensuring the
// wrapper is wired correctly and never panics.
func TestLoggingDoesNotPanic(t *testing.T) {
	Setup(true)
	Debug("debug line", "key", "value")
	Info("info line", "count", 1, "ok", true)
	Error("error line", "err", errors.New("boom"))

	Setup(false)
	Info("after switching to info level")

	// An odd number of key/value args must not panic (the trailing key is dropped).
	Info("odd args", "dangling")
}
