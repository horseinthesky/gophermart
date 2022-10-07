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
	UserExists = errors.New(`user exists`)
	UserDoesNotExist = errors.New(`user does not exist`)
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
	Id        int
	Name      string `json:"login" db:"name"`
	Password  string `json:"password" db:"-"`
	Passhash  string `db:"passhash"`
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
		Current   float64 `json:"current"`
		Withdrawn float64 `json:"withdrawn"`
	}

	Order struct {
		Number     string    `json:"number" db:"id"`
		Status     Status    `json:"status"`
		Accrual    float64   `json:"accrual"`
		UploadedAt time.Time `json:"uploaded_at"`
	}

	Withdraw struct {
		Order string  `json:"order" db:"orderid"`
		Sum   float64 `json:"sum"`
	}

	Storage interface {
		Init(context.Context) error
		Check(context.Context) error
		CreateUser(context.Context, User) (User, error)
		GetUser(context.Context, User) (User, error)
		Close()
	}
)
