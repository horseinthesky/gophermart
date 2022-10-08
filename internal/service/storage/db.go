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
	users_schema := `
		CREATE TABLE IF NOT EXISTS users (
			id serial PRIMARY KEY,
			name text NOT NULL,
			passhash text NOT NULL,
			current double precision DEFAULT 0,
			withdrawn double precision DEFAULT 0
		)
	`

	orders_schema := `
		CREATE TABLE IF NOT EXISTS orders (
			id serial PRIMARY KEY,
			userid integer NOT NULL,
			status integer NOT NULL,
			accrual double precision,
			uploaded_at timestamptz NOT NULL
		)
	`

	withdrawns_schema := `
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

	tx.ExecContext(ctx, users_schema)
	tx.ExecContext(ctx, orders_schema)
	tx.ExecContext(ctx, withdrawns_schema)

	return tx.Commit()
}

func (d *DB) Check(ctx context.Context) error {
	return d.conn.PingContext(ctx)
}

func (d *DB) CreateUser(ctx context.Context, user User) (User, error) {
	existingUser := User{}
	err := d.conn.GetContext(ctx, &existingUser, `SELECT * FROM users WHERE name=$1`, user.Name)
	if err == nil {
		return User{}, ErrUserExists
	}

	_, err = d.conn.NamedExec(`INSERT INTO users (name, passhash) VALUES (:name, :passhash)`, user)

	registeredUser := User{}
	err = d.conn.Get(&registeredUser, `SELECT * FROM users WHERE name=$1`, user.Name)

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

func (d *DB) Close() {
	d.conn.Close()
}
