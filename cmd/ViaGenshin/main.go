package main

import (
	"bufio"
	"encoding/json"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Fumeng24/ViaGenshin/internal/config"
	"github.com/Fumeng24/ViaGenshin/internal/core"
	"github.com/Fumeng24/ViaGenshin/pkg/logger"
)

var c *config.Config

func init() {
	f := os.Getenv("VIA_GENSHIN_CONFIG_FILE")
	if len(os.Args) > 1 {
		f = os.Args[1]
	}
	if f == "" {
		_, err := os.Stat("config.json")
		if err != nil {
			p, _ := json.MarshalIndent(config.DefaultConfig, "", "  ")
			logger.Warn().Msgf("VIA_GENSHIN_CONFIG_FILE not set, here is the default config:\n%s", p)
			logger.Warn().Msg("You can save it to a file named 'config.json' and run the program again")
			logger.Warn().Msg("Press 'Enter' to exit ...")
			bufio.NewReader(os.Stdin).ReadBytes('\n')
			os.Exit(0)
		}
		f = "config.json"
	}
	var err error
	c, err = config.LoadConfig(f)
	if err != nil {
		panic(err)
	}
	switch c.LogLevel {
	case "trace":
		logger.Logger = logger.Logger.Level(zerolog.TraceLevel)
	case "debug":
		logger.Logger = logger.Logger.Level(zerolog.DebugLevel)
	case "info":
		logger.Logger = logger.Logger.Level(zerolog.InfoLevel)
	case "silent", "disabled":
		logger.Logger = logger.Logger.Level(zerolog.Disabled)
	}
}

func main() {

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Logger = log.With().Caller().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.CallerFieldName = "module"
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		return filepath.Base(file)
	}
	s := core.NewService(c)

	exited := make(chan error)
	go func() {
		logger.Info().Msg("Service is starting")
		exited <- s.Start()
	}()

	// Wait for a signal to quit:
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-exited:
		if err != nil {
			logger.Error().Err(err).Msg("Service exited")
		}
	case <-sig:
		logger.Info().Msg("Signal received, stopping service")
		if err := s.Stop(); err != nil {
			logger.Error().Err(err).Msg("Service stop failed")
		}
	}
}
