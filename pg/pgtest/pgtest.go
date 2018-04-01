// Package pgtest provides utilities for creating and destroying test databases
// in PostgreSQL. It's used in eventdb's Store and E2E tests. The code is based
// on and largely identical to chain's pgtest package.
package pgtest

import (
	"context"
	"database/sql"
	"math/rand"
	"net/url"
	"runtime"
	"testing"
	"time"

	"github.com/findrandomevents/eventdb/errors"
	"github.com/lib/pq"
)

var (
	// DefaultURL is the default URL used for accessing the postgres server in open()
	DefaultURL = "postgres://localhost/postgres?sslmode=disable"
	dbpool     = make(chan *sql.DB, 4)

	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	gcDur  = 3 * time.Minute

	// DefaultSchema is a SQL query that's executed when a new database is
	// created in NewDB. You can put SQL in here that you want to be executed
	// before every test.
	DefaultSchema = `
		-- this is slow, so only do it once per db
		CREATE EXTENSION IF NOT EXISTS postgis;
	`
)

// NewDB creates a connection to a fresh PostgreSQL database for testing.
//
// Don't worry about closing the DB. It will close on its own when garbage collected.
func NewDB(t testing.TB) *sql.DB {
	t.Helper()

	runtime.GC() // give hte finalizers a chance to run

	ctx := context.Background()

	db, err := getdb(ctx, "", DefaultSchema)
	if err != nil {
		t.Fatal(err)
	}
	runtime.SetFinalizer(db, (*sql.DB).Close)

	return db
}

func open(ctx context.Context, baseURL, schema string) (*sql.DB, error) {
	if baseURL == "" {
		baseURL = DefaultURL
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	ctldb, err := sql.Open("postgres", baseURL)
	if err != nil {
		return nil, errors.E(err, "create test db")
	}
	defer ctldb.Close()

	if err = gcdbs(ctldb); err != nil {
		return nil, err
	}

	dbname := pickName("db")
	u.Path = "/" + dbname
	_, err = ctldb.Exec("CREATE DATABASE " + pq.QuoteIdentifier(dbname))
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("postgres", u.String())
	if err != nil {
		return nil, errors.E(err, "open test db")
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// NewTx creates a transaction in a fresh PostgreSQL database for testing. It's
// automatically rolled back when the Tx is garbage collected and potentially
// reused by another test.
func NewTx(t testing.TB) *sql.Tx {
	t.Helper()

	runtime.GC() // give hte finalizers a chance to run

	ctx := context.Background()

	db, err := getdb(ctx, "", DefaultSchema)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}

	runtime.SetFinalizer(tx, finaldb{db}.finalizeTx)

	return tx
}

func pickName(prefix string) (s string) {
	const chars = "abcdefghijklmnopqrstuvwxyz"
	for i := 0; i < 10; i++ {
		s += string(chars[random.Intn(len(chars))])
	}
	return formatPrefix(prefix, time.Now()) + s
}

func formatPrefix(prefix string, t time.Time) string {
	return "pgtest_" + prefix + "_" + t.UTC().Format("20060102150405") + "Z_"
}

type finaldb struct{ db *sql.DB }

func (f finaldb) finalizeTx(tx *sql.Tx) {
	go func() { // don't block the finalizer goroutine for too long
		err := tx.Rollback()
		if err != nil {
			// If the tx has been committed (or if anything
			// else goes wrong), we can't reuse db.
			f.db.Close()
			return
		}
		select {
		case dbpool <- f.db:
		default:
			f.db.Close() // pool is full
		}
	}()
}

func gcdbs(db *sql.DB) error {
	gcTime := time.Now().Add(-gcDur)
	const q = `
		SELECT datname FROM pg_database
		WHERE datname LIKE 'pgtest_%' AND datname < $1`
	rows, err := db.Query(q, formatPrefix("db", gcTime))
	if err != nil {
		return err
	}
	var names []string
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return err
		}
		names = append(names, name)
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	for i, name := range names {
		if i > 5 {
			break // drop up to five per test
		}
		go db.Exec("DROP DATABASE " + pq.QuoteIdentifier(name))
	}
	return nil
}

func getdb(ctx context.Context, url, schema string) (*sql.DB, error) {
	select {
	case db := <-dbpool:
		return db, nil
	default:
		return open(ctx, url, schema)
	}
}
