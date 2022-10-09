package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	for {
		time.Sleep(1 * time.Second)

		select {
		case <-ctx.Done():
			log.Println("accrual system poll successfully cancelled")
			return
		default:
			orders, err := s.db.GetOrders(ctx, statusesToProcess)
			if err != nil {
				log.Printf("accrual processor failed to get orders from DB: %s", err)
				continue
			}

			for _, order := range orders {
				s.processOrder(ctx, order)
			}
		}
	}
}

func (s *Service) processOrder(ctx context.Context, order storage.Order) {
	url := fmt.Sprintf("http://%s/api/orders/%s", s.config.AccrualAddress, order.Number)

	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("accrual processor failed create request for order %s: %s", order.Number, err)
		return
	}

	response, err := s.client.Do(request)
	if err != nil {
		log.Printf("accrual processor failed to process order %s: %s", order.Number, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusInternalServerError {
		log.Printf("accrual system failed to process order %s", order.Number)
		return
	}

	if response.StatusCode == http.StatusNoContent {
		log.Printf("accrual system has no order %s registered", order.Number)
		return
	}

	if response.StatusCode == http.StatusTooManyRequests {
		log.Println("accrual system is overloaded")
		time.Sleep(5 * time.Second)
		return
	}

	processedOrder := storage.Order{}
	err = json.NewDecoder(response.Body).Decode(&processedOrder)
	if err != nil {
		log.Printf("accrual processor failed to parse processed order %s: %s", order.Number, err)
		return
	}

	err = s.db.UpdateOrder(ctx, processedOrder)
	if err != nil {
		log.Printf("accrual processor failed to update order %s in DB: %s", order.Number, err)
		return
	}

	log.Printf("successfully updated order %s status", order.Number)
}
