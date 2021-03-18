package main

import "github.com/rs/zerolog"

func setupLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

}
