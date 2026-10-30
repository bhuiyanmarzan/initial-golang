package database

import (
	"context"
	"database/sql"
	"errors"
)

type DBTX interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type Transactor struct{ db *sql.DB }

func NewTransactor(db *sql.DB) *Transactor { return &Transactor{db: db} }
func (t *Transactor) Withtx(ctx context.Context, opts *sql.TxOptions, fn func(*sql.Tx) error) error {
	tx, err := t.db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// Scope must be built from verified authentication and authorized membership,
// never directly from headers, URL parameters, or an untrusted request DTO.
// The future RLS policies must use these exact app.* setting names.
type Scope struct{ UserID, OrganizationID, StoreID, CustomerID string }

func (t *Transactor) WithinScope(ctx context.Context, s Scope, opts *sql.TxOptions, fn func(*sql.Tx) error) error {
	if s.UserID == "" {
		return errors.New("verified user is required for a scoped transaction")
	}
	if s.StoreID != "" && s.OrganizationID == "" {
		return errors.New("store scope requires organization scope")
	}
	return t.Withtx(ctx, opts, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `SELECT set_config('app.user_id',$1,true),set_config('app.org_id',$2,true),set_config('app.store_id',$3,true),set_config('app.customer_id',$4,true)`, s.UserID, s.OrganizationID, s.StoreID, s.CustomerID)
		if err != nil {
			return err
		}
		return fn(tx)
	})
}
