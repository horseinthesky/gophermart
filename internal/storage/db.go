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
			id bigint PRIMARY KEY,
			name text NOT NULL,
			passhash text NOT NULL,
			current double precision,
			withdrawn double precision
		)
	`

	orders_schema := `
		CREATE TABLE IF NOT EXISTS orders (
			id text PRIMARY KEY,
			userid bigint NOT NULL,
			status integer NOT NULL,
			accrual double precision,
			uploaded_at timestamptz NOT NULL
		)
	`

	tx, err := d.conn.Beginx()
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	defer tx.Rollback()

	tx.ExecContext(ctx, users_schema)
	tx.ExecContext(ctx, orders_schema)

	return tx.Commit()
}

func (d *DB) Check(ctx context.Context) error {
	return d.conn.PingContext(ctx)
}

func (d *DB) CreateUser(ctx context.Context, user User) error {
	return nil
}

func (d *DB) Close() {
	d.conn.Close()
}
