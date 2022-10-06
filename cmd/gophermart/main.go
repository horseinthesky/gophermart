package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"log"

	"gophermart/internal"
)

func main() {
	service, err := internal.New(internal.Config{DatabaseURI: "user=postgres port=5432 sslmode=disable"})
	if err != nil {
		log.Println(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	service.Run(ctx)

	term := make(chan os.Signal)
	signal.Notify(term, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	sig := <-term
	log.Printf("signal received: %v; terminating...\n", sig)

	cancel()
	service.Stop()
}
