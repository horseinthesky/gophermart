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

func getUserNameFromRequest(r *http.Request) string {
	return r.Context().Value(contextUserNameKey).(string)
}

func validLuhn(orderNumber string) bool {
	orderNum, err := strconv.Atoi(orderNumber)
	if err != nil {
		return false
	}

	if !luhn.Valid(orderNum) {
		return false
	}

	return true
}

func (s *Service) handleNewOrder() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userName := getUserNameFromRequest(r)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.log.Errorf("failed to read request body due to: %s", err)

			http.Error(w, `failed to read payload`, http.StatusInternalServerError)
			return
		}

		orderNumberString := strings.TrimSuffix(string(body), "\n")
		if !validLuhn(orderNumberString) {
			http.Error(w, "order number is incorrect", http.StatusUnprocessableEntity)
			return
		}

		newOrder := storage.Order{
			RegisteredBy: userName,
			Number:       orderNumberString,
			UploadedAt:   time.Now(),
		}

		if err := s.db.SaveOrder(r.Context(), newOrder); err != nil {
			if errors.Is(err, storage.ErrOrderAlreadyRegisteredByUser) {
				s.log.Warnf("%s tried to register order %s but already did this earlier", userName, orderNumberString)

				w.Write([]byte(`order already registered by you`))
				return
			}

			if errors.Is(err, storage.ErrOrderAlreadyRegisteredBySomeoneElse) {
				s.log.Warnf("%s tried to register order %s but someone else already did this earlier", userName, orderNumberString)

				http.Error(w, "order already rgistered by another user", http.StatusConflict)
				return
			}

			s.log.Errorf("failed to save order to DB due to: %s", err)

			http.Error(w, `failed to register order`, http.StatusInternalServerError)
			return
		}

		s.log.Infof("order %s successfully registered by %s", orderNumberString, userName)

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`order registered`))
	})
}

func (s *Service) handleOrders() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userName := getUserNameFromRequest(r)

		orders, err := s.db.GetUserOrders(r.Context(), userName, "uploaded_at")
		if err != nil {
			s.log.Errorf("failed to get user orders from DB due to: %s", err)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to get orders"}`))
			return
		}

		res, err := json.Marshal(orders)
		if err != nil {
			s.log.Errorf("failed to marshal user orders due to: %s", err)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to marshal orders"}`))
			return
		}

		if len(orders) == 0 {
			w.WriteHeader(http.StatusNoContent)
		}

		s.log.Infof("user %s successfully got his orders", userName)

		w.Write([]byte(res))
	})
}

func (s *Service) handleBalance() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userName := getUserNameFromRequest(r)

		balance, err := s.db.GetUserBalance(r.Context(), userName)
		if err != nil {
			s.log.Errorf("failed to get user balance from DB due to: %s", err)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to get balance"}`))
		}

		res, err := json.Marshal(balance)
		if err != nil {
			s.log.Errorf("failed to marshal user balance due to: %s", err)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to marshal balance"}`))
		}

		s.log.Infof("user %s successfully got his balance", userName)

		w.Write([]byte(res))
	})
}

func (s *Service) handleWithdrawal() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userName := getUserNameFromRequest(r)

		withdrawal := storage.Withdrawal{}
		err := json.NewDecoder(r.Body).Decode(&withdrawal)
		if err != nil {
			s.log.Errorf("failed to parse withdrawal payload due to: %s", err)

			http.Error(w, `{"error": "bad or no payload"}`, http.StatusBadRequest)
			return
		}

		withdrawal.RegisteredBy = userName
		withdrawal.ProcessedAt = time.Now()

		if !validLuhn(withdrawal.Order) {
			http.Error(w, "order number is incorrect", http.StatusUnprocessableEntity)
			return
		}

		err = s.db.SaveWithdrawal(r.Context(), withdrawal)
		if err != nil {
			if errors.Is(err, storage.ErrNotEnoughPoints) {
				s.log.Warnf("user %s has not enough points to process withdrawal", userName)

				w.WriteHeader(http.StatusPaymentRequired)
				w.Write([]byte(`{"status": "error", "message": "not enough points to withdraw"}`))
				return
			}

			s.log.Errorf("failed to save withdrawal to DB due to: %s", err)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to withdraw"}`))
			return
		}

		s.log.Infof("user %s successfully withdrew some points", userName)

		w.Write([]byte(`{"status": "success", "message": "withdrawal registered"}`))
	})
}

func (s *Service) handleWithdrawals() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userName := getUserNameFromRequest(r)

		withdrawals, err := s.db.GetWithdrawals(r.Context(), userName, "processed_at")
		if err != nil {
			s.log.Errorf("failed to get withdrawals from DB due to: %s", err)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to get withdrawals"}`))
			return
		}

		if len(withdrawals) == 0 {
			s.log.Infof("no withdrawals found for user %s", userName)

			w.WriteHeader(http.StatusNoContent)
			w.Write([]byte(`{"status": "error", "message": "no withdrawals found"}`))
			return
		}

		res, err := json.Marshal(withdrawals)
		if err != nil {
			s.log.Errorf("failed to marshal withdrawals due to: %s", err)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status": "error", "message": "failed to marshal withdrawals"}`))
		}

		s.log.Infof("user %s successfully got his withdrawals", userName)

		w.Write([]byte(res))
	})
}
