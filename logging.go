package main

import (
	"github.com/aljo242/chef"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func setupLogger(cfg chef.ServerConfig) {
	if cfg.DebugLog {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("log level is DEBUG")
	} else {
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
		log.Error().Msg("log level is ERROR")

	}
}
