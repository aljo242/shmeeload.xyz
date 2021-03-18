package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func setupLogger(cfg ServerConfig) {
	//zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if cfg.DebugLog {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("log level is DEBUG")
	} else {
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		log.Error().Msg("log level is ERROR")

	}

}
