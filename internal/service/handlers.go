package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gophermart/internal/service/storage"

	"github.com/theplant/luhn"
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
		if errors.Is(err, storage.ErrUserExists) {
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
				Value: fmt.Sprint(registeredUser.ID),
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
		if errors.Is(err, storage.ErrUserDoesNotExist) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"status": "error", "message": "login/password does not exists"}`))
			return
		}

		http.SetCookie(w,
			&http.Cookie{
				Name:  "secret_id",
				Value: fmt.Sprint(registeredUser.ID),
			})

		w.Write([]byte(`{"status": "success", "message": "authenticated"}`))
	})
}

func (s *Service) handleNewOrder() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uploadedAt := time.Now()

		if r.Header.Get("Content-Type") != "text/plain" {
			http.Error(w, `Content-Type must be "text/plain"`, http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, `failed to read payload`, http.StatusInternalServerError)
			return
		}

		orderString := strings.TrimSuffix(string(body), "\n")

		orderNum, err := strconv.Atoi(orderString)
		if err != nil {
			http.Error(w, "Order number is incorrect", http.StatusBadRequest)
			return
		}

		if !luhn.Valid(orderNum) {
			http.Error(w, "Order number has wrong format.", http.StatusUnprocessableEntity)
			return
		}

		userIDValue, _ := r.Cookie("secret_id")
		userID, err := strconv.Atoi(userIDValue.Value)

		newOrder := storage.Order{
			UserID: userID,
			Number: orderString,
			UploadedAt: uploadedAt,
		}

		if err := s.db.SaveOrder(r.Context(), newOrder); err != nil {
			if errors.Is(err, storage.ErrOrderAlreadyRegisteredByUser) {
				w.Write([]byte(`order already registered by you`))
				return
			}

			if errors.Is(err, storage.ErrOrderAlreadyRegisteredBySomeoneElse) {
				http.Error(w, "order already rgistered by another user", http.StatusConflict)
				return
			}

			http.Error(w, `failed to register order`, http.StatusInternalServerError)
			return
		}

		w.Write([]byte(`order registered`))
	})
}
