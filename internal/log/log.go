// Package log is a thin wrapper over the logging backend (currently zerolog).
// The rest of the code depends only on this package, so the backend can be
// swapped in one place without touching any call sites.
package log

import (
	"os"

	"github.com/rs/zerolog"
)

var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

// Setup configures the global log level. When debug is true, debug-level lines
// are emitted; otherwise the minimum level is info.
func Setup(debug bool) {
	level := zerolog.InfoLevel
	if debug {
		level = zerolog.DebugLevel
	}
	logger = logger.Level(level)
	Info("logger configured", "level", level.String())
}

// Debug, Info, and Error emit a structured log line at the given level. Extra
// context is passed as alternating key/value pairs, e.g.
//
//	log.Error("upgrade failed", "err", err, "handler", "serveWs")
func Debug(msg string, kv ...any) { emit(logger.Debug(), msg, kv...) }
func Info(msg string, kv ...any)  { emit(logger.Info(), msg, kv...) }
func Error(msg string, kv ...any) { emit(logger.Error(), msg, kv...) }

func emit(e *zerolog.Event, msg string, kv ...any) {
	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			continue
		}
		switch v := kv[i+1].(type) {
		case error:
			e = e.AnErr(key, v)
		case string:
			e = e.Str(key, v)
		case int:
			e = e.Int(key, v)
		case bool:
			e = e.Bool(key, v)
		default:
			e = e.Interface(key, v)
		}
	}
	e.Msg(msg)
}
