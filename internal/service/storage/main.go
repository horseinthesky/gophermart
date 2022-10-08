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
		ID         int
		Number     string
		Status     Status
		Accrual    float64
		UploadedAt time.Time `db:"uploaded_at"`
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
		GetUser(context.Context, User) (User, error)
		Close()
	}
)
