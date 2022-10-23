package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gophermart/internal/service"
)

func main() {
	cfg, err := service.PrepareConfig()
	if err != nil {
		log.Fatal(fmt.Errorf("failed to prepare gophermart service config: %w", err))
	}

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
