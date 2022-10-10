package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrUserExists       = errors.New(`user exists`)
	ErrUserDoesNotExist = errors.New(`user does not exist`)

	ErrOrderAlreadyRegisteredByUser        = errors.New(`order already registered by user`)
	ErrOrderAlreadyRegisteredBySomeoneElse = errors.New(`order already registered by other user`)

	ErrNotEnoughPoints = errors.New(`user balance is too low`)
)

type Status int

const (
	StatusNew Status = iota
	StatusRegistered
	StatusProcessing
	StatusInvalid
	StatusProcessed
)

func (s Status) String() string {
	return [...]string{"NEW", "REGISTERED", "PROCESSING", "INVALID", "PROCESSED"}[s]
}

func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

var toID = map[string]Status{
	"NEW":        StatusNew,
	"REGISTERED": StatusRegistered,
	"PROCESSING": StatusProcessing,
	"INVALID":    StatusInvalid,
	"PROCESSED":  StatusProcessed,
}

func (s *Status) UnmarshalJSON(data []byte) error {
	var status string
	err := json.Unmarshal(data, &status)
	if err != nil {
		return err
	}

	id, ok := toID[status]
	if !ok {
		return errors.New("invalid value for Key")
	}

	*s = id

	return nil
}

type User struct {
	ID        int
	Name      string `json:"login"`
	Password  string
	Passhash  string
	Current   float64
	Withdrawn float64
}

func (u *User) HashPassword() {
	h := sha256.New()
	h.Write([]byte(u.Password))
	u.Passhash = hex.EncodeToString(h.Sum(nil))
}

type (
	Balance struct {
		Current   float64
		Withdrawn float64
	}

	Order struct {
		ID         int    `json:"-"`
		UserID     int    `json:"-"`
		Number     string `json:"order"`
		Status     Status
		Accrual    float64   `json:",omitempty"`
		UploadedAt time.Time `json:"uploaded_at" db:"uploaded_at"`
	}

	Withdrawal struct {
		ID          int    `json:"-"`
		UserID      int    `json:"-"`
		Order       string `db:"orderid"`
		Sum         float64
		ProcessedAt time.Time `json:"processed_at" db:"processed_at"`
	}

	Storage interface {
		Init(context.Context) error
		Check(context.Context) error

		CreateUser(context.Context, User) (User, error)
		GetUserByName(context.Context, User) (User, error)
		GetUserByID(context.Context, int) (User, error)
		GetUserBalance(context.Context, int) (Balance, error)

		SaveOrder(context.Context, Order) error
		UpdateOrder(context.Context, Order) error
		GetUserOrders(context.Context, int, string) ([]Order, error)
		GetOrders(context.Context, []Status) ([]Order, error)

		SaveWithdrawal(context.Context, Withdrawal) error
		GetWithdrawals(context.Context, int, string) ([]Withdrawal, error)

		Close()
	}
)
