package storage

import (
	"context"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type GORMDriver struct {
	conn *gorm.DB
}

func NewGORMDriver(uri string) (Storage, error) {
	db, err := gorm.Open(postgres.Open(uri), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open DB connection: %w", err)
	}

	return &GORMDriver{db}, nil
}

func (g *GORMDriver) Init(ctx context.Context) error {
	return nil
}

func (g *GORMDriver) Check(ctx context.Context) error {
	sqlDB, _ := g.conn.DB()
	return sqlDB.PingContext(ctx)
}

func (g *GORMDriver) CreateUser(ctx context.Context, user User) error {
	existingUser := User{}

	g.conn.WithContext(ctx).Where("name = ?", user.Name).Take(&existingUser)
	if existingUser.ID != 0 {
		return ErrUserExists
	}

	g.conn.WithContext(ctx).Omit("password").Create(&user)

	return nil
}

func (g *GORMDriver) GetUserByCreds(ctx context.Context, user User) (User, error) {
	g.conn.WithContext(ctx).Where("name = ? AND passhash = ?", user.Name, user.Passhash).Take(&user)
	if user.ID == 0 {
		return User{}, ErrUserDoesNotExist
	}
	return user, nil
}

func (g *GORMDriver) GetUserByName(ctx context.Context, userName string) (User, error) {
	user := User{}

	g.conn.WithContext(ctx).Where("name = ?", userName).Take(&user)
	if user.ID == 0 {
		return User{}, ErrUserDoesNotExist
	}

	return user, nil
}

func (g *GORMDriver) GetUserBalance(ctx context.Context, userName string) (Balance, error) {
	user := User{}

	g.conn.WithContext(ctx).Where("name = ?", userName).Take(&user)
	if user.ID == 0 {
		return Balance{}, ErrUserDoesNotExist
	}

	return Balance{
		Current:   user.Current,
		Withdrawn: user.Withdrawn,
	}, nil
}

func (g *GORMDriver) SaveOrder(ctx context.Context, order Order) error {
	existingOrder := Order{}

	g.conn.WithContext(ctx).Where("number = ?", order.Number).Take(&existingOrder)
	if existingOrder.ID != 0 {
		if existingOrder.RegisteredBy == order.RegisteredBy {
			return ErrOrderAlreadyRegisteredByUser
		}

		if existingOrder.RegisteredBy != order.RegisteredBy {
			return ErrOrderAlreadyRegisteredBySomeoneElse
		}
	}

	g.conn.WithContext(ctx).Create(&order)

	return nil
}

func (g *GORMDriver) UpdateOrder(ctx context.Context, order AccrualOrder) error {
	g.conn.WithContext(ctx).Model(&Order{}).Where("number = ?", order.Order).Updates(map[string]interface{}{"status": order.Status, "accrual": order.Accrual})

	updatedOrder := Order{}
	g.conn.WithContext(ctx).Where("number = ?", order.Order).Take(&updatedOrder)

	g.conn.WithContext(ctx).Model(&User{}).Where("name = ?", updatedOrder.RegisteredBy).Update("current", gorm.Expr("current + ?", updatedOrder.Accrual))

	return nil
}

func (g *GORMDriver) GetUserOrders(ctx context.Context, userName string, orderField string) ([]Order, error) {
	orders := []Order{}
	g.conn.WithContext(ctx).Order(orderField).Where("registered_by=?", userName).Find(&orders)

	return orders, nil
}

func (g *GORMDriver) GetOrders(ctx context.Context, statuses []Status) ([]Order, error) {
	orders := []Order{}
	g.conn.WithContext(ctx).Where("status IN ?", statuses).Find(&orders)

	return orders, nil
}

func (g *GORMDriver) SaveWithdrawal(ctx context.Context, withdrawal Withdrawal) error {
	user := User{}

	g.conn.WithContext(ctx).Where("name = ?", withdrawal.RegisteredBy).Take(&user)
	if user.ID == 0 {
		return ErrUserDoesNotExist
	}

	if user.Current < withdrawal.Sum {
		return ErrNotEnoughPoints
	}

	g.conn.WithContext(ctx).Model(&user).Updates(map[string]interface{}{"current": gorm.Expr("current - ?", withdrawal.Sum), "withdrawn": gorm.Expr("withdrawn + ?", withdrawal.Sum)})

	g.conn.WithContext(ctx).Create(&withdrawal)

	return nil
}

func (g *GORMDriver) GetWithdrawals(ctx context.Context, userName string, orderField string) ([]Withdrawal, error) {
	withdrawals := []Withdrawal{}
	g.conn.WithContext(ctx).Order(orderField).Where("registered_by=?", userName).Find(&withdrawals)

	return withdrawals, nil
}

func (g *GORMDriver) Close() {
	sqlDB, _ := g.conn.DB()
	sqlDB.Close()
}
