package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"gophermart/internal/service/storage"

	"github.com/theplant/luhn"
)

func (s *Service) handleRegister() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

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

		registeredUser, err := s.db.GetUserByName(r.Context(), user)
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

		orderNumberString := strings.TrimSuffix(string(body), "\n")

		orderNum, err := strconv.Atoi(orderNumberString)
		if err != nil {
			http.Error(w, "Order number is incorrect", http.StatusBadRequest)
			return
		}

		if !luhn.Valid(orderNum) {
			http.Error(w, "Order number has wrong format.", http.StatusUnprocessableEntity)
			return
		}

		userIDString, _ := r.Cookie("secret_id")
		userID, _ := strconv.Atoi(userIDString.Value)

		newOrder := storage.Order{
			UserID:     userID,
			Number:     orderNumberString,
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

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`order registered`))
	})
}

func (s *Service) handleOrders() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIDString, _ := r.Cookie("secret_id")
		userID, _ := strconv.Atoi(userIDString.Value)

		orders, err := s.db.GetUserOrders(r.Context(), userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to get orders"}`))
			return
		}

		if len(orders) == 0 {
			w.WriteHeader(http.StatusNoContent)
			w.Write([]byte(`{"status": "error", "message": "no orders found"}`))
			return
		}

		sort.Sort(storage.OrderByDate(orders))

		res, err := json.Marshal(orders)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to marshal orders"}`))
		}

		w.Write([]byte(res))
	})
}

func (s *Service) handleBalance() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIDString, _ := r.Cookie("secret_id")
		userID, _ := strconv.Atoi(userIDString.Value)

		balance, err := s.db.GetUserBalance(r.Context(), userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to get balance"}`))
		}

		res, err := json.Marshal(balance)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to marshal balance"}`))
		}

		w.Write([]byte(res))
	})
}

func (s *Service) handleWithdrawal() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIDString, _ := r.Cookie("secret_id")
		userID, _ := strconv.Atoi(userIDString.Value)

		withdrawal := storage.Withdrawal{}
		err := json.NewDecoder(r.Body).Decode(&withdrawal)
		if err != nil {
			http.Error(w, `{"error": "bad or no payload"}`, http.StatusBadRequest)
			return
		}

		withdrawal.UserID = userID
		withdrawal.ProcessedAt = time.Now()

		orderNum, _ := strconv.Atoi(withdrawal.Order)

		if !luhn.Valid(orderNum) {
			http.Error(w, "Order number has wrong format.", http.StatusUnprocessableEntity)
			return
		}

		err = s.db.SaveWithdrawal(r.Context(), withdrawal)
		if err != nil {
			if errors.Is(err, storage.ErrNotEnoughPoints) {
				w.WriteHeader(http.StatusPaymentRequired)
				w.Write([]byte(`{"status": "error", "message": "not enough points to withdraw"}`))
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to withdraw"}`))
			return
		}

		w.Write([]byte(`{"status": "success", "message": "withdrawal registered"}`))
	})
}

func (s *Service) handleWithdrawals() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIDString, _ := r.Cookie("secret_id")
		userID, _ := strconv.Atoi(userIDString.Value)

		withdrawals, err := s.db.GetWithdrawals(r.Context(), userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to get withdrawals"}`))
			return
		}

		if len(withdrawals) == 0 {
			w.WriteHeader(http.StatusNoContent)
			w.Write([]byte(`{"status": "error", "message": "no withdrawals found"}`))
			return
		}

		sort.Sort(storage.WithdrawalsByDate(withdrawals))

		res, err := json.Marshal(withdrawals)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to marshal withdrawals"}`))
		}

		w.Write([]byte(res))
	})
}
