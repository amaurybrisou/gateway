package config

import (
	"fmt"
	"os"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

	BindAddr string `env:"BIND_ADDR" envDefault:"0.0.0.0"`
	BindPort int    `env:"BIND_PORT" envDefault:"8080"`
}

func New() Config {
	envFile := ".env"
	if envVar, ok := os.LookupEnv("ENV"); ok {
		envFile = fmt.Sprintf("%s.%s", envFile, envVar)
	}

	err := godotenv.Load(".env", envFile)
	if err != nil {
		log.Warn().Err(err).Msg("could not load config, using default values")
	}

	c := Config{}
	err = env.Parse(&c)
	if err != nil {
		log.Fatal().Err(err).Msg("could not load config")
	}

	return c
}
