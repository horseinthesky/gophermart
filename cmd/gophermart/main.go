package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"log"

	"gophermart/internal/service"
)

func main() {
	service, err := service.New(service.Config{DatabaseURI: "user=postgres port=5432 sslmode=disable"})
	if err != nil {
		log.Println(err)
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
