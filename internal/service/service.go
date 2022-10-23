package service

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"gophermart/internal/service/storage"
	"gophermart/internal/service/token"
)

type Service struct {
	config Config
	router *chi.Mux
	db     storage.Storage
	client *http.Client
	tm     token.Maker
	log    *zap.SugaredLogger
	wg     sync.WaitGroup
}

func New(cfg Config) (*Service, error) {
	// db, err := storage.NewSQLxDriver(cfg.DatabaseURI)
	db, err := storage.NewGORMDriver(cfg.DatabaseURI)
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

	logger, err := initLogger(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		return nil, err
	}

	return &Service{cfg, nil, db, client, tokenMaker, logger, sync.WaitGroup{}}, nil
}

func (s *Service) Run(ctx context.Context) {
	s.setupRouter()

	err := s.db.Init(ctx)
	if err != nil {
		s.log.Fatalf("failed to init DB: %s", err)
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.processOrders(ctx)
	}()

	s.log.Infof("gophermart server started at: %s; debug=%v", s.config.RunAddress, s.config.Debug)
	s.log.Fatalf("server crashed due to: %s", http.ListenAndServe(s.config.RunAddress, s.router))
}

func (s *Service) Stop() {
	s.log.Infof("shutting down...")

	s.db.Close()
	s.log.Infof("connection to database closed")

	s.wg.Wait()
	s.log.Infof("successfully shut down")
}
