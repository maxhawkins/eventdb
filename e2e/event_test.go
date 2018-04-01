package e2e

import (
	"context"
	"testing"

	"github.com/findrandomevents/eventdb"
	"github.com/findrandomevents/eventdb/errors"
	"github.com/findrandomevents/eventdb/rest/client"
)

func TestEventSubmitAnonymous(t *testing.T) {
	t.Parallel()

	srv := stubServer(t)
	defer srv.Close()

	ctx := context.Background()

	client := client.New("") // anonymous
	client.BaseURL = srv.URL

	err := client.Events.Submit(ctx, eventdb.EventSubmitRequest{EventIDs: []eventdb.EventID{"1"}})
	if !errors.Is(errors.Permission, err) {
		t.Fatalf("anon user Events.Submit got %v, want %v", err, errors.Permission)
	}
}
