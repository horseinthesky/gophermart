package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gophermart/internal/service/token"
)

type ctxKey int

const (
	contextUserNameKey ctxKey = iota
)

func (s *Service) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.log.Infof("got %s request from %s for %s", r.Method, r.RemoteAddr, r.URL.Path)

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			s.log.Errorf("failed to read request body: %s", err)
			r.Body.Close()
			next.ServeHTTP(w, r)
		}
		defer r.Body.Close()

		s.log.Infof("payload: %s", string(bodyBytes))

		headersMsg := []string{}
		for header, values := range r.Header {
			headersMsg = append(headersMsg, fmt.Sprintf("%s: %s", header, strings.Join(values, ", ")))
		}
		s.log.Infof("headers: %s", strings.Join(headersMsg, "; "))

		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		next.ServeHTTP(w, r)
	})
}

func (s *Service) loginRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		tokenCookie, err := r.Cookie("token")
		if err != nil {
			if err == http.ErrNoCookie {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"status": "error", "message": "not authenticated"}`))
				return
			}

			s.log.Errorf("failed to get token from cookie due to: %s", err)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to parse cookies"}`))
			return
		}

		payload, err := s.tm.VerifyToken(tokenCookie.Value)
		if err != nil {
			if errors.Is(err, token.ErrInvalidToken) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"status": "error", "message": "invalid token"}`))
				return
			}

			if errors.Is(err, token.ErrExpiredToken) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"status": "error", "message": "expired token"}`))
				return
			}
		}

		user, err := s.db.GetUserByName(r.Context(), payload.Username)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"status": "error", "message": "user not found"}`))
			return
		}

		ctx := context.WithValue(r.Context(), contextUserNameKey, user.Name)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
