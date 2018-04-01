package pg

import (
	"context"
	"testing"
	"time"

	"github.com/findrandomevents/eventdb"
	"github.com/findrandomevents/eventdb/errors"
	"github.com/findrandomevents/eventdb/pg/pgtest"
	"github.com/go-test/deep"
)

func TestUserUpdate(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := pgtest.NewDB(t)
	store := &UserStore{DB: db}
	if err := store.Init(ctx); err != nil {
		t.Fatal(err)
	}

	const userID = "user1"

	_, err := store.GetByID(ctx, userID)
	if got, want := err, errors.E(errors.NotExist); !errors.Match(got, want) {
		t.Fatalf("GetByID error=%v, want %v", got, want)
	}

	// Mask updates the token and not other stuff
	updated, err := store.Update(ctx, userID, eventdb.UserUpdate{
		FacebookToken: "fbtok2",
		Mask:          "facebookToken",
	})
	if err != nil {
		t.Fatalf("Update(): %v", err)
	}
	if got, want := updated.FacebookToken, "fbtok2"; got != want {
		t.Fatalf("updated.FacebookToken = %v, want %v", got, want)
	}
	if got, want := updated.FacebookID, ""; got != want {
		t.Fatalf("updated.FacebookID = %v, want %v", got, want)
	}

	expected := eventdb.User{
		ID:            userID,
		Birthday:      time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC),
		TimeZone:      "UTC",
		FacebookID:    "fbid",
		FacebookToken: "fbtok",
	}

	updated, err = store.Update(ctx, userID, eventdb.UserUpdate{
		Birthday:      time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC),
		TimeZone:      "UTC",
		FacebookID:    "fbid",
		FacebookToken: "fbtok",
		Mask:          "birthday,timeZone,facebookID,facebookToken",
	})
	if err != nil {
		t.Fatalf("Update(): %v", err)
	}
	if diff := deep.Equal(updated, expected); diff != nil {
		t.Fatalf("Update() != expectedUser; %v", diff)
	}

	got, err := store.GetByID(ctx, userID)
	if err != nil {
		t.Fatalf("GetByID(): %v", err)
	}
	if diff := deep.Equal(got, expected); diff != nil {
		t.Fatalf("GetByID() != expectedUser; %v", diff)
	}
}

func TestRandomFBToken(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := pgtest.NewDB(t)
	store := &UserStore{DB: db}
	if err := store.Init(ctx); err != nil {
		t.Fatal(err)
	}

	savedID := eventdb.UserID("user1")
	savedToken := "faketoken"

	_, err := store.Update(ctx, savedID, eventdb.UserUpdate{
		FacebookToken: savedToken,
		Mask:          "facebookToken",
	})
	if err != nil {
		t.Fatal(err)
	}

	userID, token, err := store.RandomFBToken(ctx)
	if err != nil {
		t.Fatalf("RandomFBToken(): %v", err)
	}

	if got, want := token, savedToken; got != want {
		t.Fatalf("RandomFBToken() = %q, want %q", got, want)
	}
	if got, want := userID, savedID; got != want {
		t.Fatalf("RandomFBToken() userID = %q, want %q", got, want)
	}
}
