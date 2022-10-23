package storage

import (
	"context"
	"fmt"
)

type Storage interface {
	Init(context.Context) error
	Check(context.Context) error

	CreateUser(context.Context, User) error
	GetUserByCreds(context.Context, User) (User, error)
	GetUserByName(context.Context, string) (User, error)
	GetUserBalance(context.Context, string) (Balance, error)

	SaveOrder(context.Context, Order) error
	UpdateOrder(context.Context, AccrualOrder) error
	GetUserOrders(context.Context, string, string) ([]Order, error)
	GetOrders(context.Context, []Status) ([]Order, error)

	SaveWithdrawal(context.Context, Withdrawal) error
	GetWithdrawals(context.Context, string, string) ([]Withdrawal, error)

	Close()
}

var storageMap = map[string]func(string) (Storage, error){
	"sqlx": NewSQLxDriver,
	"gorm": NewSQLxDriver,
}

func NewStorage(name, uri string) (Storage, error) {
	driverCreator, ok := storageMap[name]
	if !ok {
		return nil, fmt.Errorf(`DB ORM "%s" is not supported; use "gorm/sqlx"`, name)
	}

	driver, err := driverCreator(uri)
	if err != nil {
		return nil, err
	}

	return driver, nil
}
