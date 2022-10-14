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
	"gophermart/internal/service/token"
)

type (
	Config struct {
		RunAddress     string        `env:"RUN_ADDRESS" envDefault:"localhost:8000"`
		DatabaseURI    string        `env:"DATABASE_URI" envDefault:"postgresql://postgres@localhost:5432?sslmode=disable"`
		AccrualAddress string        `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:"http://localhost:8080"`
		TokenEngine    string        `env:"TOKEN_ENGINE" envDefault:"paseto"`
		TokenDuration  time.Duration `env:"TOKEN_DURATION" envDefault:"24h"`
		Key            string        `env:"SECRET" envDefault:"cuzyouwillneverknowthissecretkey"`
		Debug          bool
	}

	Service struct {
		config Config
		router *chi.Mux
		db     storage.Storage
		client *http.Client
		tm     token.Maker
		wg     sync.WaitGroup
	}
)

func New(cfg Config) (*Service, error) {
	db, err := storage.NewDB(cfg.DatabaseURI)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	tokenMaker, err := token.NewTokenMaker(cfg.TokenEngine, cfg.Key)
	if err != nil {
		return nil, err
	}

	return &Service{cfg, nil, db, client, tokenMaker, sync.WaitGroup{}}, nil
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
