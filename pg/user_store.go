package pg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/findrandomevents/eventdb"
	"github.com/findrandomevents/eventdb/errors"
)

// UserStore stores metadata about users in a PostgreSQL database.
type UserStore struct {
	DB *sql.DB
}

// Init sets up the database schema and creates indices.
func (u *UserStore) Init(ctx context.Context) error {
	const op errors.Op = "UserStore.Init"

	_, err := u.DB.ExecContext(ctx, `
	CREATE EXTENSION IF NOT EXISTS pgcrypto;
	CREATE EXTENSION IF NOT EXISTS postgis;

    CREATE TABLE IF NOT EXISTS users (
	   sequence          SERIAL        NOT NULL,
	   user_id           TEXT,

	   birthday          DATE,
	   time_zone         VARCHAR(255),

	   facebook_id       TEXT,
	   facebook_token    TEXT
	);
	CREATE UNIQUE INDEX IF NOT EXISTS user_id_idx ON users (user_id);
	CREATE INDEX IF NOT EXISTS facebook_id_idx ON users (facebook_id);

	CREATE UNIQUE INDEX IF NOT EXISTS user_token_idx
	ON users (sequence)
	WHERE facebook_token != '';
	`)
	if err != nil {
		return errors.E(op, pgErr(err))
	}

	return nil
}

// RandomFBToken returns the Facebook OAuth token for a random user in the database
func (u *UserStore) RandomFBToken(ctx context.Context) (userID eventdb.UserID, token string, err error) {
	err = u.DB.QueryRowContext(ctx, `
		SELECT user_id, facebook_token
		FROM users
		WHERE LENGTH(facebook_token) > 0
		ORDER BY sequence
		LIMIT 1
		OFFSET floor(
			random() * (SELECT COUNT(*) FROM users WHERE LENGTH(facebook_token) > 0)
		)`).Scan(&userID, &token)
	if err == sql.ErrNoRows {
		return eventdb.UserID(userID), token, errors.E("no facebook tokens available", pgErr(err))
	}
	if err != nil {
		return eventdb.UserID(userID), token, pgErr(err)
	}

	return eventdb.UserID(userID), token, nil
}

// Update applies a UserUpdate to the given User, then returns the result.
func (u *UserStore) Update(ctx context.Context, userID eventdb.UserID, update eventdb.UserUpdate) (eventdb.User, error) {
	fields := []string{"user_id"}
	args := []interface{}{userID}

	for _, field := range strings.Split(update.Mask, ",") {
		switch field {
		case "timeZone":
			fields = append(fields, "time_zone")
			args = append(args, update.TimeZone)

		case "facebookID":
			fields = append(fields, "facebook_id")
			args = append(args, update.FacebookID)

		case "facebookToken":
			fields = append(fields, "facebook_token")
			args = append(args, update.FacebookToken)

		case "birthday":
			fields = append(fields, "birthday")
			args = append(args, update.Birthday)
		}
	}

	var placeholders []string
	for i := range fields {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
	}

	var updates []string
	for i, field := range fields {
		if i == 0 { // skip id
			continue
		}
		updates = append(updates, fmt.Sprintf("%s = $%d", field, i+1))
	}

	query := fmt.Sprintf(`
		INSERT INTO users(%s) VALUES(%s)`,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "))
	if len(updates) > 0 {
		query += " ON CONFLICT (user_id) DO UPDATE SET " + strings.Join(updates, ", ")
	}

	_, err := u.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return eventdb.User{}, pgErr(err)
	}

	user, err := u.GetByID(ctx, userID)
	if err != nil {
		return eventdb.User{}, pgErr(err)
	}

	return user, nil
}

// GetByID retrieves a User by ID.
func (u *UserStore) GetByID(ctx context.Context, userID eventdb.UserID) (eventdb.User, error) {
	var user eventdb.User

	err := u.DB.QueryRowContext(ctx, `
		SELECT
			COALESCE(user_id, ''),
			COALESCE(birthday, '0001-01-01'),
			COALESCE(facebook_id, ''),
			COALESCE(facebook_token, ''),
			COALESCE(time_zone, '')
		FROM users
		WHERE user_id = $1
	`, userID).Scan(
		&user.ID,
		&user.Birthday,
		&user.FacebookID,
		&user.FacebookToken,
		&user.TimeZone,
	)
	if err != nil {
		return user, pgErr(err)
	}

	return user, nil
}
