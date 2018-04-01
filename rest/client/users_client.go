package client

import (
	"context"

	"github.com/findrandomevents/eventdb"
)

// UsersClient provides access to the eventdb /users endpoint
type UsersClient struct {
	client *Client
}

// Update lets users update their profile data.
func (c *UsersClient) Update(ctx context.Context, id string, update eventdb.UserUpdate) (eventdb.User, error) {
	var resp eventdb.User
	if err := c.client.doJSON(ctx, "PATCH", "/users/"+id, update, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// Get retrieves User records.
func (c *UsersClient) Get(ctx context.Context, id string) (eventdb.User, error) {
	var resp eventdb.User
	if err := c.client.doJSON(ctx, "GET", "/users/"+id, nil, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}
