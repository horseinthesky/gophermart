package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	"gophermart/internal/storage"
)

type (
	Config struct {
		Address        string
		DatabaseURI    string
		AccrualAddress string
	}

	Service struct {
		router *chi.Mux
		db     storage.Storage
		wg     sync.WaitGroup
	}
)

func New(cfg Config) (*Service, error) {
	router := chi.NewRouter()

	db, err := storage.NewDB(cfg.DatabaseURI)
	if err != nil {
		return nil, err
	}

	return &Service{router, db, sync.WaitGroup{}}, nil
}

func (s *Service) Run(ctx context.Context) {
	s.setupRouter()

	err := s.db.Init(ctx)
	if err != nil {
		log.Fatalf("failed to init DB: %s", err)
	}

	log.Println(fmt.Errorf("server crashed due to %w", http.ListenAndServe("localhost:8080", s.router)))
}

func (s *Service) Stop() {
	log.Println("shutting down...")

	s.db.Close()
	log.Println("connection to database closed")

	s.wg.Wait()
	log.Println("successfully shut down")
}
