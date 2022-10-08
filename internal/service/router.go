package service

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (s *Service) setupRouter() {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(handleGzip)
	if s.config.Debug {
		r.Use(logRequest)
	}

	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", s.handleRegister())
		r.Post("/login", s.handleLogin())

		r.Group(func(r chi.Router) {
			r.Use(loginRequired)

			r.Post("/orders", s.handleNewOrder())
			r.Get("/orders", s.handleOrders())
		})
	})

	s.router = r
}
