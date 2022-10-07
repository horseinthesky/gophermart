package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"gophermart/internal/service/storage"
)

func (s *Service) handleRegister() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		body, err := ioutil.ReadAll(r.Body)
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
		if errors.Is(err, storage.UserExists) {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(`{"status": "error", "message": "user already exists"}`))
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to register user"}`))
			return
		}

		http.SetCookie(w,
			&http.Cookie{
				Name:  "secret_id",
				Value: fmt.Sprint(registeredUser.Id),
			})

		w.Write([]byte(`{"status": "success", "message": "authenticated"}`))
	})
}

func (s *Service) handleLogin() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		body, err := ioutil.ReadAll(r.Body)
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

		registeredUser, err := s.db.GetUser(r.Context(), user)
		if errors.Is(err, storage.UserDoesNotExist) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"status": "error", "message": "login/password does not exists"}`))
			return
		}

		http.SetCookie(w,
			&http.Cookie{
				Name:  "secret_id",
				Value: fmt.Sprint(registeredUser.Id),
			})

		w.Write([]byte(`{"status": "success", "message": "authenticated"}`))
	})
}
