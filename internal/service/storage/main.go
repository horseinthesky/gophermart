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
	ErrUserExists                          = errors.New(`user exists`)
	ErrUserDoesNotExist                    = errors.New(`user does not exist`)
	ErrOrderAlreadyRegisteredByUser        = errors.New(`order already registered by user`)
	ErrOrderAlreadyRegisteredBySomeoneElse = errors.New(`order already registered by other user`)
)

type Status int

const (
	New Status = iota
	Processing
	Invalid
	Processed
)

func (s Status) String() string {
	return [...]string{"NEW", "PROCESSING", "INVALID", "PROCESSED"}[s]
}

func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
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
		ID         int `json:"-"`
		UserID     int `json:"-"`
		Number     string
		Status     Status
		Accrual    float64   `json:",omitempty"`
		UploadedAt time.Time `json:"uploaded_at" db:"uploaded_at"`
	}

	Withdraw struct {
		Order       string `db:"orderid"`
		Sum         float64
		ProcessedAt time.Time `db:"processed_at"`
	}

	Storage interface {
		Init(context.Context) error
		Check(context.Context) error
		CreateUser(context.Context, User) (User, error)
		GetUserByName(context.Context, User) (User, error)
		GetUserByID(context.Context, int) (User, error)
		SaveOrder(context.Context, Order) error
		GetOrders(context.Context, int) ([]Order, error)
		GetUserBalance(context.Context, int) (Balance, error)
		Close()
	}
)

type OrderByDate []Order

func (o OrderByDate) Len() int {
	return len(o)
}

func (o OrderByDate) Less(i, j int) bool {
	return o[i].UploadedAt.Before(o[j].UploadedAt)
}

func (o OrderByDate) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}
