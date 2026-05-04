package mysql

import (
	"context"
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type DB struct {
	sql *sql.DB
}

func NewDB(dsn string) (*DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return &DB{sql: db}, nil
}

func (d *DB) Ping(ctx context.Context) error {
	return d.sql.PingContext(ctx)
}

func (d *DB) SQL() *sql.DB {
	return d.sql
}
