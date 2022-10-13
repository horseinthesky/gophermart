package service

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gophermart/internal/service/storage"

	"github.com/theplant/luhn"
)

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

		userIDString, _ := r.Cookie("user_id")
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
		w.Header().Set("Content-Type", "application/json")

		userIDString, _ := r.Cookie("user_id")
		userID, _ := strconv.Atoi(userIDString.Value)

		orders, err := s.db.GetUserOrders(r.Context(), userID, "uploaded_at")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to get orders"}`))
			return
		}

		res, err := json.Marshal(orders)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to marshal orders"}`))
			return
		}

		if len(orders) == 0 {
			w.WriteHeader(http.StatusNoContent)
		}

		w.Write([]byte(res))
	})
}

func (s *Service) handleBalance() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userIDString, _ := r.Cookie("user_id")
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
		w.Header().Set("Content-Type", "application/json")

		userIDString, _ := r.Cookie("user_id")
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
		w.Header().Set("Content-Type", "application/json")

		userIDString, _ := r.Cookie("user_id")
		userID, _ := strconv.Atoi(userIDString.Value)

		withdrawals, err := s.db.GetWithdrawals(r.Context(), userID, "processed_at")
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

		res, err := json.Marshal(withdrawals)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to marshal withdrawals"}`))
		}

		w.Write([]byte(res))
	})
}
