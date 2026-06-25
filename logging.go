package main

import (
	"github.com/aljo242/chef"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func setupLogger(cfg chef.ServerConfig) {
	if cfg.DebugLog {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	log.Info().Str("level", zerolog.GlobalLevel().String()).Msg("logger configured")
}
