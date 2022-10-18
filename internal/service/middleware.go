package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"gophermart/internal/service/token"
)

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Got %s request from %s for %s", r.Method, r.RemoteAddr, r.URL.Path)

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println("Body: failed to read")
			r.Body.Close()
			next.ServeHTTP(w, r)
		}
		defer r.Body.Close()

		log.Print("Body:", string(bodyBytes))
		log.Print("Headers:")
		for header, values := range r.Header {
			log.Print(header, values)
		}

		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		next.ServeHTTP(w, r)
	})
}

func handleGzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
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

		ctx := context.WithValue(r.Context(), "user", user.Name)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
