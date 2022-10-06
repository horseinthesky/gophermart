package internal

import (
	"github.com/go-chi/chi/v5/middleware"
)

func (s *Service) setupRouter() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
}
