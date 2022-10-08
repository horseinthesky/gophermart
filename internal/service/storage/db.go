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
			name text NOT NULL,
			passhash text NOT NULL,
			current double precision DEFAULT 0,
			withdrawn double precision DEFAULT 0
		)
	`

	ordersTable := `
		CREATE TABLE IF NOT EXISTS orders (
			id serial PRIMARY KEY,
			userid integer NOT NULL,
			number text NOT NULL,
			status int NOT NULL,
			accrual double precision DEFAULT 0,
			uploaded_at timestamptz NOT NULL
		)
	`

	withdrawnTable := `
		CREATE TABLE IF NOT EXISTS withdrawns (
			id serial PRIMARY KEY,
			userid integer NOT NULL,
			orderid integer NOT NULL,
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
	tx.ExecContext(ctx, withdrawnTable)

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

func (d *DB) GetUser(ctx context.Context, user User) (User, error) {
	existingUser := User{}
	err := d.conn.GetContext(ctx, &existingUser, `SELECT * FROM users WHERE name=$1 AND passhash=$2`, user.Name, user.Passhash)
	if err != nil {
		return User{}, ErrUserDoesNotExist
	}

	return existingUser, nil
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
