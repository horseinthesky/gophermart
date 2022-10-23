package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gophermart/internal/service/storage"
)

var statusesToProcess = []storage.Status{
	storage.StatusNew,
	storage.StatusRegistered,
	storage.StatusProcessing,
}

func (s *Service) processOrders(ctx context.Context) {
	s.log.Infof("accrual processor started")

	for {
		time.Sleep(1 * time.Second)

		select {
		case <-ctx.Done():
			s.log.Infof("accrual processor stopped")
			return
		default:
			orders, err := s.db.GetOrders(ctx, statusesToProcess)
			if err != nil {
				s.log.Errorf("accrual processor failed to get orders from DB: %s", err)
				continue
			}

			if len(orders) == 0 {
				s.log.Infof("accrual processor has no orders to process")
				continue
			}

			for _, order := range orders {
				s.processOrder(ctx, order)
			}
		}
	}
}

func (s *Service) processOrder(ctx context.Context, order storage.Order) {
	url := fmt.Sprintf("%s/api/orders/%s", s.config.AccrualAddress, order.Number)

	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		s.log.Errorf("accrual processor failed create request for order %s: %s", order.Number, err)
		return
	}

	response, err := s.client.Do(request)
	if err != nil {
		s.log.Errorf("accrual processor failed to process order %s: %s", order.Number, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusInternalServerError {
		s.log.Errorf("accrual system failed to process order %s", order.Number)
		return
	}

	if response.StatusCode == http.StatusNoContent {
		s.log.Infof("accrual system has no order %s", order.Number)
		return
	}

	if response.StatusCode == http.StatusTooManyRequests {
		s.log.Infof("accrual system is overloaded")
		time.Sleep(5 * time.Second)
		return
	}

	processedOrder := storage.AccrualOrder{}
	err = json.NewDecoder(response.Body).Decode(&processedOrder)
	if err != nil {
		s.log.Errorf("accrual processor failed to parse processed order %s: %s", order.Number, err)
		return
	}

	err = s.db.UpdateOrder(ctx, processedOrder)
	if err != nil {
		s.log.Errorf("accrual processor failed to update order %s in DB: %s", order.Number, err)
		return
	}

	s.log.Infof("successfully updated order %s status", order.Number)
}
