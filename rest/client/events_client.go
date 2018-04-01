package client

import (
	"context"

	"github.com/findrandomevents/eventdb"
)

// EventsClient provides access to the eventdb /events endpoint
type EventsClient struct {
	client *Client
}

// Search queries the database for events matching the EventSearchRequest
// and returns Event objects for the matching results.
func (c *EventsClient) Search(ctx context.Context, req eventdb.EventSearchRequest) ([]eventdb.Event, error) {
	var resp []eventdb.Event
	if err := c.client.doJSON(ctx, "POST", "/events/search", req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// Submit downloads the events using the Facebook API and saves them to the
// EventStore. It uses a random user's Facebook API token to fetch the event
// so some users must be logged in with Facebook for this method to work.
func (c *EventsClient) Submit(ctx context.Context, req eventdb.EventSubmitRequest) error {
	return c.client.doJSON(ctx, "POST", "/events", req, nil)
}
