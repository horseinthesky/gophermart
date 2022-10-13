package service

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/golang-jwt/jwt/v4"

	"gophermart/internal/service/storage"
)

var jwtKey = []byte("youwillneverknow")

type jwtClaims struct {
	jwt.StandardClaims
	Username string
}

func newJWToken(user storage.User) string {
	claims := jwtClaims{
		Username: user.Name,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)

	tokenString, _ := token.SignedString(jwtKey)

	return tokenString

}

func (s *Service) handleRegister() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to read payload"}`))
			return
		}

		user := storage.User{}
		err = json.Unmarshal(body, &user)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status": "error", "message": "failed to parse payload"}`))
			return
		}

		user.HashPassword()

		registeredUser, err := s.db.CreateUser(r.Context(), user)
		if err != nil {
			if errors.Is(err, storage.ErrUserExists) {
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"status": "error", "message": "user already exists"}`))
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to register user"}`))
			return
		}

		token := newJWToken(registeredUser)

		http.SetCookie(w,
			&http.Cookie{
				Name:  "token",
				Value: token,
			})

		w.Write([]byte(`{"status": "success", "message": "authenticated"}`))
	})
}

func (s *Service) handleLogin() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to read payload"}`))
			return
		}

		user := storage.User{}
		err = json.Unmarshal(body, &user)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status": "error", "message": "failed to parse payload"}`))
			return
		}

		user.HashPassword()

		registeredUser, err := s.db.GetUserByCreds(r.Context(), user)
		if errors.Is(err, storage.ErrUserDoesNotExist) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"status": "error", "message": "login/password does not exists"}`))
			return
		}

		token := newJWToken(registeredUser)

		http.SetCookie(w,
			&http.Cookie{
				Name:  "token",
				Value: token,
			})

		w.Write([]byte(`{"status": "success", "message": "authenticated"}`))
	})
}
