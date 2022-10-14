package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gophermart/internal/service"

	"github.com/caarlos0/env/v6"
)

func getConfig() service.Config {
	cfg := service.Config{}

	if err := env.Parse(&cfg); err != nil {
		log.Fatal(fmt.Errorf("failed to parse env vars: %w", err))
	}

	flag.StringVar(&cfg.RunAddress, "a", cfg.RunAddress, "Socket to listen on")
	flag.StringVar(&cfg.DatabaseURI, "d", cfg.DatabaseURI, "Database URI")
	flag.StringVar(&cfg.AccrualAddress, "r", cfg.AccrualAddress, "Accrual system address")
	flag.StringVar(&cfg.TokenEngine, "e", cfg.TokenEngine, "Token engine: jwt/paseto(default)")
	flag.DurationVar(&cfg.TokenDuration, "t", cfg.TokenDuration, "Token duration")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "Secret key")
	flag.BoolVar(&cfg.Debug, "D", false, "Debug mode")
	flag.Parse()

	return cfg
}

func main() {
	cfg := getConfig()

	service, err := service.New(cfg)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create gophermart service: %w", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	go service.Run(ctx)

	term := make(chan os.Signal, 1)
	signal.Notify(term, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	sig := <-term
	log.Printf("signal received: %v; terminating...\n", sig)

	cancel()
	service.Stop()
}
