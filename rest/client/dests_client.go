package client

import (
	"context"
	"fmt"

	"github.com/findrandomevents/eventdb"
)

// DestsClient provides access to the eventdb /dests endpoint
type DestsClient struct {
	client *Client
}

// Generate finds a new random event near the user's location and returns
// a DestGenerateReply that includes the new event and whether or not the search
// was successful.
func (c *DestsClient) Generate(ctx context.Context, opts eventdb.DestGenerateRequest) (eventdb.DestGenerateReply, error) {
	endpoint := fmt.Sprintf("/dests/generate?lat=%f&lng=%f", opts.Lat, opts.Lng)
	var resp eventdb.DestGenerateReply
	if err := c.client.doJSON(ctx, "POST", endpoint, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// Get retrieves a Dest from the database.
func (c *DestsClient) Get(ctx context.Context, id eventdb.DestID) (eventdb.Dest, error) {
	var resp eventdb.Dest
	if err := c.client.doJSON(ctx, "GET", "/dests/"+string(id), nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// Update updates a Dest with the user's feedback
func (c *DestsClient) Update(ctx context.Context, id eventdb.DestID, update eventdb.DestUpdate) (eventdb.Dest, error) {
	var resp eventdb.Dest
	if err := c.client.doJSON(ctx, "PATCH", "/dests/"+string(id), update, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// List lists a user's Dests by creation date.
func (c *DestsClient) List(ctx context.Context, id eventdb.DestID, update eventdb.DestUpdate) ([]eventdb.Dest, error) {
	var resp []eventdb.Dest
	if err := c.client.doJSON(ctx, "GET", "/dests", nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}
