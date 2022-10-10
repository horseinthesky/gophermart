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

	"github.com/caarlos0/env"
)

func getConfig() service.Config {
	cfg := service.Config{}

	if err := env.Parse(&cfg); err != nil {
		log.Fatal(fmt.Errorf("failed to parse env vars: %w", err))
	}

	flag.StringVar(&cfg.RunAddress, "a", cfg.RunAddress, "Socket to listen on")
	flag.StringVar(&cfg.DatabaseURI, "d", cfg.DatabaseURI, "Database URI")
	flag.StringVar(&cfg.AccrualAddress, "r", cfg.AccrualAddress, "Accrual system address")
	flag.BoolVar(&cfg.Debug, "D", false, "Debug mode")
	flag.Parse()

	return cfg
}

func main() {
	// service, err := service.New(service.Config{DatabaseURI: "user=postgres sslmode=disable"})
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
