package storage

import (
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

var toString = map[Status]string{
	StatusNew:        "NEW",
	StatusRegistered: "REGISTERED",
	StatusProcessing: "PROCESSING",
	StatusInvalid:    "INVALID",
	StatusProcessed:  "PROCESSED",
}

var toID = map[string]Status{
	"NEW":        StatusNew,
	"REGISTERED": StatusRegistered,
	"PROCESSING": StatusProcessing,
	"INVALID":    StatusInvalid,
	"PROCESSED":  StatusProcessed,
}

func (s Status) String() string {
	return toString[s]
}

func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *Status) UnmarshalJSON(data []byte) error {
	var status string
	err := json.Unmarshal(data, &status)
	if err != nil {
		return err
	}

	id, ok := toID[status]
	if !ok {
		return errors.New("invalid value for key")
	}

	*s = id

	return nil
}

type User struct {
	ID        int
	Name      string  `json:"login" gorm:"not null;unique"`
	Password  string  `gorm:"-"`
	Passhash  string  `gorm:"not null"`
	Current   float32 `gorm:"type:float8;default:0"`
	Withdrawn float32 `gorm:"type:float8;default:0"`
}

func (u *User) HashPassword() {
	h := sha256.New()
	h.Write([]byte(u.Password))
	u.Passhash = hex.EncodeToString(h.Sum(nil))
}

type (
	Balance struct {
		Current   float32
		Withdrawn float32
	}

	Order struct {
		ID           int       `json:"-"`
		RegisteredBy string    `json:"-" db:"registered_by" gorm:"not null;unique"`
		Number       string    `json:"number" gorm:"not null"`
		Status       Status    `json:"status" gorm:"not null"`
		Accrual      float32   `json:"accrual,omitempty" gorm:"type:float8;default:0"`
		UploadedAt   time.Time `json:"uploaded_at" db:"uploaded_at"`
	}

	AccrualOrder struct {
		Order   string  `json:"order"`
		Status  Status  `json:"status"`
		Accrual float32 `json:"accrual,omitempty"`
	}

	Withdrawal struct {
		ID           int       `json:"-"`
		RegisteredBy string    `json:"-" db:"registered_by" gorm:"not null;unique"`
		Order        string    `json:"order" db:"orderid" gorm:"column:orderid;not null"`
		Sum          float32   `json:"sum" gorm:"type:float8;default:0"`
		ProcessedAt  time.Time `json:"processed_at" db:"processed_at"`
	}
)
