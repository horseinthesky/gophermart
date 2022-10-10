package service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"gophermart/internal/service/storage"
)

type (
	Config struct {
		RunAddress     string `env:"RUN_ADDRESS" envDefault:"localhost:8000"`
		DatabaseURI    string `env:"DATABASE_URI" envDefault:"postgresql://postgres@localhost:5432?sslmode=disable"`
		AccrualAddress string `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:"localhost:8080"`
		Debug          bool
	}

	Service struct {
		config Config
		router *chi.Mux
		db     storage.Storage
		client *http.Client
		wg     sync.WaitGroup
	}
)

func New(cfg Config) (*Service, error) {
	db, err := storage.NewDB(cfg.DatabaseURI)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 50 * time.Millisecond,
	}

	return &Service{cfg, nil, db, client, sync.WaitGroup{}}, nil
}

func (s *Service) Run(ctx context.Context) {
	s.setupRouter()

	err := s.db.Init(ctx)
	if err != nil {
		log.Fatalf("failed to init DB: %s", err)
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.processOrders(ctx)
	}()

	log.Printf("gophermart server started at: %s; debug=%v", s.config.RunAddress, s.config.Debug)
	log.Fatal(fmt.Errorf("server crashed due to: %w", http.ListenAndServe(s.config.RunAddress, s.router)))
}

func (s *Service) Stop() {
	log.Println("shutting down...")

	s.db.Close()
	log.Println("connection to database closed")

	s.wg.Wait()
	log.Println("successfully shut down")
}
