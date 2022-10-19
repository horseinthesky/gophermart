package service

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"gophermart/internal/service/storage"
)

func (s *Service) handleRegister() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.log.Errorf("failed to read request body due to: %s", err)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to read payload"}`))
			return
		}

		user := storage.User{}
		err = json.Unmarshal(body, &user)
		if err != nil {
			s.log.Errorf("failed to parse user data from payload due to: %s", err)

			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status": "error", "message": "failed to parse payload"}`))
			return
		}

		user.HashPassword()

		registeredUser, err := s.db.CreateUser(r.Context(), user)
		if err != nil {
			if errors.Is(err, storage.ErrUserExists) {
				s.log.Warnf("user %s tried to register but already exists in DB", user.Name)

				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"status": "error", "message": "user already exists"}`))
				return
			}

			s.log.Errorf("failed to save user to DB due to: %s", err)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to register user"}`))
			return
		}

		token, _, err := s.tm.CreateToken(registeredUser.Name, s.config.TokenDuration)
		if err != nil {
			s.log.Errorf("failed to create new token due to: %s", err)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to create token"}`))
			return
		}

		http.SetCookie(w,
			&http.Cookie{
				Name:  "token",
				Value: token,
			})

		s.log.Infof("user %s successfully registered", registeredUser.Name)

		w.Write([]byte(`{"status": "success", "message": "authenticated"}`))
	})
}

func (s *Service) handleLogin() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.log.Errorf("failed to read request body due to: %s", err)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to read payload"}`))
			return
		}

		user := storage.User{}
		err = json.Unmarshal(body, &user)
		if err != nil {
			s.log.Errorf("failed to parse user data from payload due to: %s", err)

			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status": "error", "message": "failed to parse payload"}`))
			return
		}

		user.HashPassword()

		registeredUser, err := s.db.GetUserByCreds(r.Context(), user)
		if errors.Is(err, storage.ErrUserDoesNotExist) {
			s.log.Warnf("user %s tried to log in but not found in DB", user.Name)

			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"status": "error", "message": "login/password pair does not exists"}`))
			return
		}

		token, _, err := s.tm.CreateToken(registeredUser.Name, s.config.TokenDuration)
		if err != nil {
			s.log.Errorf("failed to create new token due to: %s", err)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to create token"}`))
			return
		}

		http.SetCookie(w,
			&http.Cookie{
				Name:  "token",
				Value: token,
			})

		s.log.Infof("user %s successfully logged in", registeredUser.Name)

		w.Write([]byte(`{"status": "success", "message": "authenticated"}`))
	})
}
