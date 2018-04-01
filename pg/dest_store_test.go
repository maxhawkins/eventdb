package pg

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-test/deep"

	"github.com/findrandomevents/eventdb"
	"github.com/findrandomevents/eventdb/pg/pgtest"
)

func TestDestStoreInit(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbx := pgtest.NewDB(t)
	destStore := &DestStore{DB: dbx}
	if err := destStore.Init(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestDestStoreList(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbx := pgtest.NewDB(t)
	destStore := &DestStore{DB: dbx}
	if err := destStore.Init(ctx); err != nil {
		t.Fatalf("DestStore.Init: %v", err)
	}

	var savedDests []eventdb.Dest

	for i := 0; i < 15; i++ {
		dest, err := destStore.Create(ctx, eventdb.Dest{
			UserID:  "user1",
			EventID: eventdb.EventID(fmt.Sprintf("event-%d", i)),
		})
		if err != nil {
			t.Fatalf("DestStore.Create: %v", err)
		}

		savedDests = append([]eventdb.Dest{dest}, savedDests...)
	}

	dests, err := destStore.ListForUser(ctx, "user1", eventdb.DestListRequest{})
	if err != nil {
		t.Fatalf("DestStore.ListForUser: %v", err)
	}

	expected := savedDests[:10]
	if diff := deep.Equal(dests, expected); diff != nil {
		t.Fatalf("DestStore.List(); %v", diff)
	}
}

func TestDestStoreUpdate(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbx := pgtest.NewDB(t)
	destStore := &DestStore{DB: dbx}
	if err := destStore.Init(ctx); err != nil {
		t.Fatalf("DestStore.Init: %v", err)
	}

	dest, err := destStore.Create(ctx, eventdb.Dest{
		UserID:  "user1",
		EventID: "event1",
	})
	if err != nil {
		t.Fatalf("DestStore.Create: %v", err)
	}

	status := "new status"
	feedback := "new feedback"
	updated, err := destStore.Update(ctx, dest.ID, eventdb.DestUpdate{
		Status:   status,
		Feedback: feedback,
		Mask:     "feedback,status",
	})
	if err != nil {
		t.Fatalf("DestStore.Update: %v", err)
	}

	if got, want := updated.Status, status; got != want {
		t.Fatalf("updated: got status %q, want %q", got, want)
	}
	if got, want := updated.Feedback, feedback; got != want {
		t.Fatalf("updated: got feedback %q, want %q", got, want)
	}
}
