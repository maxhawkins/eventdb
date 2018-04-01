package pg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/findrandomevents/eventdb"
	"github.com/findrandomevents/eventdb/errors"
)

// DestStore stores and retrives Dests from a PostgreSQL database.
type DestStore struct {
	DB *sql.DB
}

// Init sets up the database schema.
func (s *DestStore) Init(ctx context.Context) error {
	const op errors.Op = "DestStore.Init"

	_, err := s.DB.ExecContext(ctx, `
    CREATE TABLE IF NOT EXISTS dests (
	   sequence       SERIAL        NOT NULL,
	   id             VARCHAR(40),

	   user_id        VARCHAR(40)   NOT NULL,
	   event_id       VARCHAR(40)   NOT NULL,

     feedback       TEXT,
     status         TEXT,

	   created_at     TIMESTAMP     NOT NULL DEFAULT NOW()
	);
	CREATE UNIQUE INDEX IF NOT EXISTS dest_id_idx ON dests (id);`)
	if err != nil {
		return errors.E(op, pgErr(err))
	}

	return nil
}

// Create saves a new Dest
func (s *DestStore) Create(ctx context.Context, dest eventdb.Dest) (eventdb.Dest, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return dest, err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `
	INSERT INTO dests
		(user_id, event_id)
	VALUES
		($1, $2)
	RETURNING sequence`, dest.UserID, dest.EventID)

	var sequence int64
	if err = row.Scan(&sequence); err != nil {
		return dest, errors.E(pgErr(err), "get dest id")
	}

	destID := eventdb.DestID(fmt.Sprint(sequence))
	_, err = tx.ExecContext(ctx, `
	UPDATE dests
	SET id = $1
	WHERE sequence = $2`, destID, sequence)
	if err != nil {
		return dest, errors.E(pgErr(err), "set dest hash id")
	}

	if err := tx.Commit(); err != nil {
		return dest, pgErr(err)
	}

	return s.Get(ctx, destID)
}

// Get retrieves a Dest by ID.
func (s *DestStore) Get(ctx context.Context, id eventdb.DestID) (eventdb.Dest, error) {
	dests, err := s.list(ctx, "WHERE id = $1", id)
	if err != nil {
		return eventdb.Dest{}, err
	}
	if len(dests) == 0 {
		return eventdb.Dest{}, errors.E(errors.NotExist, "dest not found")
	}

	dest := dests[0]
	return dest, nil
}

// Update applies a DestUpdate to the given Dest, then returns the result.
func (s *DestStore) Update(ctx context.Context, id eventdb.DestID, update eventdb.DestUpdate) (eventdb.Dest, error) {
	fields := []string{"id"}
	args := []interface{}{id}

	for _, field := range strings.Split(update.Mask, ",") {
		switch field {
		case "feedback":
			fields = append(fields, "feedback")
			args = append(args, update.Feedback)

		case "status":
			fields = append(fields, "status")
			args = append(args, update.Status)
		}
	}

	var updates []string
	for i, field := range fields {
		if i == 0 { // skip id field
			continue
		}
		updates = append(updates, fmt.Sprintf("%s = $%d", field, i+1))
	}

	query := fmt.Sprintf(`
		UPDATE dests SET %s WHERE id = $1`,
		strings.Join(updates, ", "))
	_, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return eventdb.Dest{}, pgErr(err)
	}

	dest, err := s.Get(ctx, id)
	if err != nil {
		return eventdb.Dest{}, pgErr(err)
	}

	return dest, nil
}

// ListForUser returns all of a user's dests, ordered by creation date.
func (s *DestStore) ListForUser(ctx context.Context, userID eventdb.UserID, opts eventdb.DestListRequest) ([]eventdb.Dest, error) {
	const pageSize = 10

	offset := opts.Page * pageSize
	limit := pageSize

	return s.list(ctx, `
		WHERE user_id = $1
		ORDER BY created_at DESC
		OFFSET $2
		LIMIT $3
		`, userID, offset, limit)
}

func (s *DestStore) list(ctx context.Context, expr string, vals ...interface{}) ([]eventdb.Dest, error) {
	query := fmt.Sprintf(`
	SELECT
		id,
		user_id,
		event_id,
		COALESCE(feedback, ''),
		COALESCE(status, ''),
		created_at
	FROM dests
	%s`, expr)

	rows, err := s.DB.QueryContext(ctx, query, vals...)
	if err != nil {
		return nil, errors.E(pgErr(err), "dest list")
	}
	defer rows.Close()

	dests := []eventdb.Dest{}
	for rows.Next() {
		var dest eventdb.Dest
		err := rows.Scan(
			&dest.ID,
			&dest.UserID,
			&dest.EventID,
			&dest.Feedback,
			&dest.Status,
			&dest.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		dests = append(dests, dest)
	}

	return dests, nil
}
