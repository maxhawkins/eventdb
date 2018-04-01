package pg

import (
	"context"
	"database/sql"

	"github.com/findrandomevents/eventdb/errors"
	"github.com/lib/pq"
)

// pgErr converts an error produced by lib/pq into an eventdb domain error.
// All sql statements in package pg should return errors wrapped by pgErr.
func pgErr(err error) error {
	if err == sql.ErrNoRows {
		return errors.E(errors.NotExist)
	}

	e, ok := err.(*pq.Error)
	if !ok {
		return err
	}

	switch e.Code.Name() {
	case "unique_violation":
		return errors.E(errors.Exist, e.Message)
	case "query_canceled":
		return errors.E(context.Canceled)
	default:
		return e
	}
}
