package service

import (
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
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

		userIDString, err := r.Cookie("secret_id")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"status": "error", "message": "not authenticated"}`))
			return
		}

		userID, err := strconv.Atoi(userIDString.Value)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status": "error", "message": "invalid secret_id"}`))
			return
		}

		_, err = s.db.GetUserBalance(r.Context(), userID)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"status": "error", "message": "not authenticated"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
