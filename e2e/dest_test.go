package e2e

import (
	"context"
	"testing"

	"github.com/findrandomevents/eventdb"
	"github.com/findrandomevents/eventdb/errors"
	"github.com/findrandomevents/eventdb/rest/client"
)

func TestGenerateDest(t *testing.T) {
	t.Parallel()

	srv := stubServer(t)
	defer srv.Close()

	client := client.New("user")
	client.BaseURL = srv.URL

	ctx := context.Background()

	// First, add some events to the database
	savedEventIDs := []eventdb.EventID{
		"1", "2", "3", "4", "5",
	}
	err := client.Events.Submit(ctx, eventdb.EventSubmitRequest{
		EventIDs: savedEventIDs,
	})
	if err != nil {
		t.Fatal("submit events: ", err)
	}

	// Then choose a random one.
	reply, err := client.Dests.Generate(ctx, eventdb.DestGenerateRequest{
		Lat: 45.962815043539,
		Lng: 15.485937595367,
	})
	if err != nil {
		t.Fatal("generate dest: ", err)
	}

	// You should get a result:
	if got, want := reply.Result, eventdb.GenerateOK; got != want {
		t.Fatalf("generate got result %q, want %q", got, want)
	}

	if len(reply.Dests) == 0 {
		t.Fatalf("returned no dests")
	}
	eventID := reply.Dests[0].EventID

	if len(reply.Events) == 0 {
		t.Fatalf("returned no events")
	}
	event := reply.Events[0]

	if got, want := event.ID, eventID; got != want {
		t.Fatalf("sideloaded event with id = %q, want id = %q", got, want)
	}

	// And that result should be one of the ones we submitted earlier.
	var match bool
	for _, savedID := range savedEventIDs {
		if eventID == savedID {
			match = true
			break
		}
	}
	if !match {
		t.Fatalf("chose event id %q, want an id in %v", eventID, savedEventIDs)
	}
}

func TestGenerateDestTooFast(t *testing.T) {
	t.Parallel()

	srv := stubServer(t)
	defer srv.Close()

	client := client.New("user")
	client.BaseURL = srv.URL

	ctx := context.Background()

	// First, add some events to the database
	savedEventIDs := []eventdb.EventID{
		"1", "2", "3", "4", "5",
	}
	err := client.Events.Submit(ctx, eventdb.EventSubmitRequest{
		EventIDs: savedEventIDs,
	})
	if err != nil {
		t.Fatal("submit events: ", err)
	}

	reply, err := client.Dests.Generate(ctx, eventdb.DestGenerateRequest{
		Lat: 45.962815043539,
		Lng: 15.485937595367,
	})
	if err != nil {
		t.Fatal("generate dest: ", err)
	}
	if got, want := reply.Result, eventdb.GenerateOK; got != want {
		t.Fatalf("generate got result %q, want %q", got, want)
	}
	if len(reply.Dests) == 0 {
		t.Fatalf("returned no dests")
	}

	reply, err = client.Dests.Generate(ctx, eventdb.DestGenerateRequest{
		Lat: 45.962815043539,
		Lng: 15.485937595367,
	})
	if err != nil {
		t.Fatal("generate dest: ", err)
	}
	if got, want := reply.Result, eventdb.GenerateWait; got != want {
		t.Fatalf("generate got result %q, want %q", got, want)
	}
	if len(reply.Dests) == 0 {
		t.Fatalf("returned no dests")
	}
}

func TestNoNewEvents(t *testing.T) {
	t.Parallel()

	srv := stubServer(t)
	defer srv.Close()

	client := client.New("user")
	client.BaseURL = srv.URL

	ctx := context.Background()

	resp, err := client.Dests.Generate(ctx, eventdb.DestGenerateRequest{
		Lat: 0,
		Lng: 0,
	})
	if err != nil {
		t.Fatalf("Dests.Generate: %v", err)
	}

	if got, want := resp.Result, eventdb.GenerateNoResults; got != want {
		t.Fatalf("Generate status=%q, want %q", got, want)
	}
}

func TestUpdateStrangerEvent(t *testing.T) {
	t.Parallel()

	srv := stubServer(t)
	defer srv.Close()

	ctx := context.Background()

	// First, some stranger makes a dest
	strangerClient := client.New("stranger")
	strangerClient.BaseURL = srv.URL

	err := strangerClient.Events.Submit(ctx, eventdb.EventSubmitRequest{
		EventIDs: []eventdb.EventID{"dummyevent"},
	})
	if err != nil {
		t.Fatal("submit events: ", err)
	}

	reply, err := strangerClient.Dests.Generate(ctx, eventdb.DestGenerateRequest{
		Lat: 45.962815043539,
		Lng: 15.485937595367,
	})
	if err != nil {
		t.Fatal("generate dest: ", err)
	}
	if got, want := reply.Result, eventdb.GenerateOK; got != want {
		t.Fatalf("generate result=%v, want %v", got, want)
	}
	if got, want := len(reply.Dests), 1; got != want {
		t.Fatalf("generate created %d dests, want %d", got, want)
	}
	dest := reply.Dests[0]

	// Then we (maliciously) try to access it

	client := client.New("user")
	client.BaseURL = srv.URL

	_, err = client.Dests.Get(ctx, dest.ID)
	if got, kind := err, errors.Permission; !errors.Is(kind, err) {
		t.Fatalf("get stranger's dest returned %v, want %v", got, kind)
	}

	_, err = client.Dests.Update(ctx, dest.ID, eventdb.DestUpdate{
		Status: "pwned",
		Mask:   "status",
	})
	if got, kind := err, errors.Permission; !errors.Is(kind, err) {
		t.Fatalf("get stranger's dest returned %v, want %v", got, kind)
	}
}
