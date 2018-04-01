package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/findrandomevents/eventdb/errors"
)

// Client provides a client to eventdb's REST API.
//
// Don't construct a Client directly. Use New() instead.
type Client struct {
	// HTTP is the underlying HTTP client used send requests.
	HTTP *http.Client
	// BaseURL is the HTTP endpoint for the REST API. Can be overridden for tests.
	// It defaults to https://backend.findrandomevents.com
	BaseURL string
	// JWT is the user credential used to authenticate with eventdb.
	//
	// We're using Firebase auth, so this must be retrieved from the Firebase API.
	JWT string

	Users  *UsersClient
	Events *EventsClient
	Dests  *DestsClient
}

// New constructs a new Client
func New(jwt string) *Client {
	client := &Client{
		HTTP:    http.DefaultClient,
		BaseURL: "https://backend.findrandomevents.com",
		JWT:     jwt,
	}

	client.Users = &UsersClient{client}
	client.Events = &EventsClient{client}
	client.Dests = &DestsClient{client}

	return client
}

func (c Client) doJSON(ctx context.Context, method, path string, req interface{}, resp interface{}) error {
	var reqBody io.Reader
	if req != nil {
		reqJS, err := json.Marshal(req)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(reqJS)
	}

	r, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return err
	}
	r = r.WithContext(ctx)

	if c.JWT != "" {
		r.Header.Set("Authorization", "Bearer "+c.JWT)
	}

	w, err := c.HTTP.Do(r)
	if err != nil {
		return err
	}
	defer w.Body.Close()

	if status := w.StatusCode; status != http.StatusOK {
		var resp errors.Response
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			return err
		}
		return resp.ToError()
	}

	if resp != nil {
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			return err
		}
	}

	return nil
}
