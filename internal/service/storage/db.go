package storage

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DB struct {
	conn *sqlx.DB
}

func NewDB(uri string) (Storage, error) {
	conn, err := sqlx.Open("postgres", uri)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB connection: %w", err)
	}

	return &DB{conn}, nil
}

func (d *DB) Init(ctx context.Context) error {
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
			userid integer NOT NULL REFERENCES users (id) ON DELETE CASCADE,
			number text NOT NULL,
			status int NOT NULL,
			accrual double precision DEFAULT 0,
			uploaded_at timestamptz NOT NULL
		)
	`

	withdrawalsTable := `
		CREATE TABLE IF NOT EXISTS withdrawals (
			id serial PRIMARY KEY,
			userid integer NOT NULL REFERENCES users (id) ON DELETE CASCADE,
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

func (d *DB) Check(ctx context.Context) error {
	return d.conn.PingContext(ctx)
}

func (d *DB) Close() {
	d.conn.Close()
}

func (d *DB) CreateUser(ctx context.Context, user User) (User, error) {
	existingUser := User{}

	err := d.conn.GetContext(ctx, &existingUser, `SELECT * FROM users WHERE name=$1`, user.Name)
	if err == nil {
		return User{}, ErrUserExists
	}

	_, err = d.conn.NamedExecContext(ctx, `INSERT INTO users (name, passhash) VALUES (:name, :passhash)`, user)
	if err != nil {
		return User{}, fmt.Errorf("failed to insert new user: %w", err)
	}

	registeredUser := User{}
	err = d.conn.GetContext(ctx, &registeredUser, `SELECT * FROM users WHERE name=$1`, user.Name)

	return registeredUser, err
}

func (d *DB) GetUserByCreds(ctx context.Context, user User) (User, error) {
	existingUser := User{}

	err := d.conn.GetContext(ctx, &existingUser, `SELECT * FROM users WHERE name=$1 AND passhash=$2`, user.Name, user.Passhash)
	if err != nil {
		return User{}, ErrUserDoesNotExist
	}

	return existingUser, nil
}

func (d *DB) GetUserByName(ctx context.Context, userName string) (User, error) {
	user := User{}

	err := d.conn.GetContext(ctx, &user, `SELECT * FROM users WHERE name=$1`, userName)
	if err != nil {
		return User{}, ErrUserDoesNotExist
	}

	return user, nil
}

func (d *DB) SaveOrder(ctx context.Context, order Order) error {
	existingOrder := Order{}

	err := d.conn.GetContext(ctx, &existingOrder, `SELECT * FROM orders WHERE number=$1`, order.Number)
	if err == nil {
		if existingOrder.UserID == order.UserID {
			return ErrOrderAlreadyRegisteredByUser
		}

		if existingOrder.UserID != order.UserID {
			return ErrOrderAlreadyRegisteredBySomeoneElse
		}
	}

	_, err = d.conn.NamedExecContext(ctx, `INSERT INTO orders (userid, number, status, uploaded_at) VALUES (:userid, :number, :status, :uploaded_at)`, order)
	if err != nil {
		return fmt.Errorf("failed to insert new order: %w", err)
	}

	return nil
}

func (d *DB) UpdateOrder(ctx context.Context, order AccrualOrder) error {
	_, err := d.conn.NamedExecContext(ctx, `UPDATE orders SET status = :status, accrual = :accrual WHERE number=:order`, order)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	updatedOrder := Order{}
	err = d.conn.GetContext(ctx, &updatedOrder, `SELECT * FROM orders WHERE number=$1`, order.Order)
	if err != nil {
		return fmt.Errorf("failed to get updated order: %w", err)
	}

	_, err = d.conn.NamedExecContext(ctx, `UPDATE users SET current = users.current + :accrual WHERE id=:userid`, updatedOrder)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	return nil
}

func (d *DB) GetUserOrders(ctx context.Context, userID int, orderField string) ([]Order, error) {
	query := fmt.Sprintf(`SELECT * FROM orders WHERE userid=$1 ORDER BY %s`, orderField)

	orders := []Order{}

	err := d.conn.SelectContext(ctx, &orders, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders")
	}

	return orders, nil
}

func (d *DB) GetOrders(ctx context.Context, statuses []Status) ([]Order, error) {
	// sqlx.In returns queries with the `?` bindvar, we can rebind it for our backend
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

func (d *DB) GetUserBalance(ctx context.Context, userID int) (Balance, error) {
	user := User{}

	err := d.conn.GetContext(ctx, &user, `SELECT * FROM users WHERE id=$1`, userID)
	if err != nil {
		return Balance{}, fmt.Errorf("failed to get user balance")
	}

	return Balance{
		Current:   user.Current,
		Withdrawn: user.Withdrawn,
	}, nil
}

func (d *DB) SaveWithdrawal(ctx context.Context, withdrawal Withdrawal) error {
	user := User{}

	err := d.conn.GetContext(ctx, &user, `SELECT * FROM users WHERE id=$1`, withdrawal.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user to withdraw from")
	}

	if user.Current < withdrawal.Sum {
		return ErrNotEnoughPoints
	}

	_, err = d.conn.NamedExecContext(ctx, `UPDATE users SET current = users.current - :sum, withdrawn = users.withdrawn + :sum WHERE id = :userid`, withdrawal)
	if err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	_, err = d.conn.NamedExecContext(ctx, `INSERT INTO withdrawals (userid, orderid, sum, processed_at) VALUES (:userid, :orderid, :sum, :processed_at)`, withdrawal)
	if err != nil {
		return fmt.Errorf("failed to insert new withdraw: %w", err)
	}

	return nil
}

func (d *DB) GetWithdrawals(ctx context.Context, userID int, orderField string) ([]Withdrawal, error) {
	query := fmt.Sprintf(`SELECT * FROM withdrawals WHERE userid=$1 ORDER BY %s`, orderField)

	withdrawals := []Withdrawal{}

	err := d.conn.SelectContext(ctx, &withdrawals, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get withdrawals")
	}

	return withdrawals, nil
}
