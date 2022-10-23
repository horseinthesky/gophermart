package storage

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type SQLxDriver struct {
	conn *sqlx.DB
}

func NewSQLxDriver(uri string) (Storage, error) {
	conn, err := sqlx.Open("postgres", uri)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB connection: %w", err)
	}

	return &SQLxDriver{conn}, nil
}

func (d *SQLxDriver) Init(ctx context.Context) error {
	usersTable := `
		CREATE TABLE IF NOT EXISTS users (
			id serial PRIMARY KEY,
			name text NOT NULL UNIQUE,
			passhash text NOT NULL,
			current double precision DEFAULT 0,
			withdrawn double precision DEFAULT 0
		)
	`

	ordersTable := `
		CREATE TABLE IF NOT EXISTS orders (
			id serial PRIMARY KEY,
			registered_by text NOT NULL REFERENCES users (name) ON DELETE CASCADE,
			number text NOT NULL UNIQUE,
			status int NOT NULL,
			accrual double precision DEFAULT 0,
			uploaded_at timestamptz NOT NULL
		)
	`

	withdrawalsTable := `
		CREATE TABLE IF NOT EXISTS withdrawals (
			id serial PRIMARY KEY,
			registered_by text NOT NULL REFERENCES users (name) ON DELETE CASCADE,
			orderid text NOT NULL,
			sum double precision NOT NULL,
			processed_at timestamptz NOT NULL
		)
	`

	tx, err := d.conn.Beginx()
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	defer tx.Rollback()

	tx.ExecContext(ctx, usersTable)
	tx.ExecContext(ctx, ordersTable)
	tx.ExecContext(ctx, withdrawalsTable)

	return tx.Commit()
}

func (d *SQLxDriver) Check(ctx context.Context) error {
	return d.conn.PingContext(ctx)
}

func (d *SQLxDriver) Close() {
	d.conn.Close()
}

func (d *SQLxDriver) CreateUser(ctx context.Context, user User) error {
	existingUser := User{}

	err := d.conn.GetContext(ctx, &existingUser, `SELECT * FROM users WHERE name=$1`, user.Name)
	if err == nil {
		return ErrUserExists
	}

	_, err = d.conn.NamedExecContext(ctx, `INSERT INTO users (name, passhash) VALUES (:name, :passhash)`, user)
	if err != nil {
		return fmt.Errorf("failed to insert new user: %w", err)
	}

	return nil
}

func (d *SQLxDriver) GetUserByCreds(ctx context.Context, user User) (User, error) {
	existingUser := User{}

	err := d.conn.GetContext(ctx, &existingUser, `SELECT * FROM users WHERE name=$1 AND passhash=$2`, user.Name, user.Passhash)
	if err != nil {
		return User{}, ErrUserDoesNotExist
	}

	return existingUser, nil
}

func (d *SQLxDriver) GetUserByName(ctx context.Context, userName string) (User, error) {
	user := User{}

	err := d.conn.GetContext(ctx, &user, `SELECT * FROM users WHERE name=$1`, userName)
	if err != nil {
		return User{}, ErrUserDoesNotExist
	}

	return user, nil
}

func (d *SQLxDriver) SaveOrder(ctx context.Context, order Order) error {
	existingOrder := Order{}

	err := d.conn.GetContext(ctx, &existingOrder, `SELECT * FROM orders WHERE number=$1`, order.Number)
	if err == nil {
		if existingOrder.RegisteredBy == order.RegisteredBy {
			return ErrOrderAlreadyRegisteredByUser
		}

		if existingOrder.RegisteredBy != order.RegisteredBy {
			return ErrOrderAlreadyRegisteredBySomeoneElse
		}
	}

	_, err = d.conn.NamedExecContext(ctx, `INSERT INTO orders (registered_by, number, status, uploaded_at) VALUES (:registered_by, :number, :status, :uploaded_at)`, order)
	if err != nil {
		return fmt.Errorf("failed to insert new order: %w", err)
	}

	return nil
}

func (d *SQLxDriver) UpdateOrder(ctx context.Context, order AccrualOrder) error {
	_, err := d.conn.NamedExecContext(ctx, `UPDATE orders SET status = :status, accrual = :accrual WHERE number=:order`, order)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	updatedOrder := Order{}
	err = d.conn.GetContext(ctx, &updatedOrder, `SELECT * FROM orders WHERE number=$1`, order.Order)
	if err != nil {
		return fmt.Errorf("failed to get updated order: %w", err)
	}

	_, err = d.conn.NamedExecContext(ctx, `UPDATE users SET current = users.current + :accrual WHERE name=:registered_by`, updatedOrder)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	return nil
}

func (d *SQLxDriver) GetUserOrders(ctx context.Context, userName string, orderField string) ([]Order, error) {
	query := fmt.Sprintf(`SELECT * FROM orders WHERE registered_by=$1 ORDER BY %s`, orderField)

	orders := []Order{}

	err := d.conn.SelectContext(ctx, &orders, query, userName)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders")
	}

	return orders, nil
}

func (d *SQLxDriver) GetOrders(ctx context.Context, statuses []Status) ([]Order, error) {
	query, args, err := sqlx.In(`SELECT * FROM orders WHERE status IN (?)`, statuses)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare IN query: %w", err)
	}

	query = d.conn.Rebind(query)

	orders := []Order{}

	err = d.conn.SelectContext(ctx, &orders, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	return orders, nil
}

func (d *SQLxDriver) GetUserBalance(ctx context.Context, userName string) (Balance, error) {
	user := User{}

	err := d.conn.GetContext(ctx, &user, `SELECT * FROM users WHERE name=$1`, userName)
	if err != nil {
		return Balance{}, fmt.Errorf("failed to get user balance")
	}

	return Balance{
		Current:   user.Current,
		Withdrawn: user.Withdrawn,
	}, nil
}

func (d *SQLxDriver) SaveWithdrawal(ctx context.Context, withdrawal Withdrawal) error {
	user := User{}

	err := d.conn.GetContext(ctx, &user, `SELECT * FROM users WHERE name=$1`, withdrawal.RegisteredBy)
	if err != nil {
		return ErrUserDoesNotExist
	}

	if user.Current < withdrawal.Sum {
		return ErrNotEnoughPoints
	}

	_, err = d.conn.NamedExecContext(ctx, `UPDATE users SET current = users.current - :sum, withdrawn = users.withdrawn + :sum WHERE name = :registered_by`, withdrawal)
	if err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	_, err = d.conn.NamedExecContext(ctx, `INSERT INTO withdrawals (registered_by, orderid, sum, processed_at) VALUES (:registered_by, :orderid, :sum, :processed_at)`, withdrawal)
	if err != nil {
		return fmt.Errorf("failed to insert new withdraw: %w", err)
	}

	return nil
}

func (d *SQLxDriver) GetWithdrawals(ctx context.Context, userName string, orderField string) ([]Withdrawal, error) {
	query := fmt.Sprintf(`SELECT * FROM withdrawals WHERE registered_by=$1 ORDER BY %s`, orderField)

	withdrawals := []Withdrawal{}

	err := d.conn.SelectContext(ctx, &withdrawals, query, userName)
	if err != nil {
		return nil, fmt.Errorf("failed to get withdrawals")
	}

	return withdrawals, nil
}
