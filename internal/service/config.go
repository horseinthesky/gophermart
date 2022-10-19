package service

import (
	"flag"
	"fmt"
	"time"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	RunAddress     string        `env:"RUN_ADDRESS" envDefault:"localhost:8000"`
	DatabaseURI    string        `env:"DATABASE_URI" envDefault:"postgresql://postgres@localhost:5432?sslmode=disable"`
	AccrualAddress string        `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:"http://localhost:8080"`
	TokenEngine    string        `env:"TOKEN_ENGINE" envDefault:"paseto"`
	TokenDuration  time.Duration `env:"TOKEN_DURATION" envDefault:"24h"`
	Key            string        `env:"SECRET" envDefault:"cuzyouwillneverknowthissecretkey"`
	LogLevel       string        `env:"LOG_LEVEL" envDefault:"error"`
	LogFormat      string        `env:"LOG_FORMAT" envDefault:"printf"`
	Debug          bool
}

func PrepareConfig() (Config, error) {
	cfg := Config{}

	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse env vars: %w", err)
	}

	flag.StringVar(&cfg.RunAddress, "a", cfg.RunAddress, "Socket to listen on")
	flag.StringVar(&cfg.DatabaseURI, "d", cfg.DatabaseURI, "Database URI")
	flag.StringVar(&cfg.AccrualAddress, "r", cfg.AccrualAddress, "Accrual system address")
	flag.StringVar(&cfg.TokenEngine, "e", cfg.TokenEngine, "Token engine: jwt/paseto")
	flag.DurationVar(&cfg.TokenDuration, "t", cfg.TokenDuration, "Token duration")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "Secret key")
	flag.StringVar(&cfg.LogLevel, "l", cfg.LogLevel, "Log level: debug/info/warn/error")
	flag.StringVar(&cfg.LogFormat, "f", cfg.LogFormat, "Log foramt: json/printf")
	flag.BoolVar(&cfg.Debug, "D", false, "Debug mode")
	flag.Parse()

	return cfg, nil
}
