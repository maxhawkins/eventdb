// Package e2e contains end-to-end tests for the eventdb package. They test from
// the rest interface all the way down to the database layer.
package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/findrandomevents/eventdb"
	"github.com/findrandomevents/eventdb/auth"
	"github.com/findrandomevents/eventdb/pg"
	"github.com/findrandomevents/eventdb/pg/pgtest"
	"github.com/findrandomevents/eventdb/rest"
	"github.com/findrandomevents/eventdb/service"
)

// stubServer starts a new httptest.Server with a stubbed out eventdb service.
// You must call Close on the returned server after you're done with it.
func stubServer(t *testing.T) *httptest.Server {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	service := stubService(ctx, t)
	handler := rest.New(service)

	return httptest.NewServer(handler)
}

// stubService returns an eventdb Service where all the external dependencies
// have been stubbed out, and the database is backed by a pgtest temp db.
func stubService(ctx context.Context, t *testing.T) *service.Service {
	db := pgtest.NewDB(t)

	userStore := &pg.UserStore{DB: db}
	if err := userStore.Init(ctx); err != nil {
		t.Fatal(err)
	}

	// Add a dummy user with a facebook token
	_, err := userStore.Update(ctx, "dummy", eventdb.UserUpdate{
		FacebookID:    "dummy-id",
		FacebookToken: "dummy-token",
		Mask:          "facebookID,facebookToken",
	})
	if err != nil {
		t.Fatal(err)
	}

	eventStore := &pg.EventStore{DB: db}
	if err := eventStore.Init(ctx); err != nil {
		t.Fatal(err)
	}

	destStore := &pg.DestStore{DB: db}
	if err := destStore.Init(ctx); err != nil {
		t.Fatal(err)
	}
	srv := &service.Service{
		UserStore:  userStore,
		DestStore:  destStore,
		EventStore: eventStore,

		FacebookClient: func(string) service.FacebookClient {
			return stubFacebookClient{}
		},
		Time: stubTime(time.Date(2017, 8, 17, 14, 0, 0, 0, time.UTC)),

		Auth: stubAuth{},
	}

	return srv
}

// stubFacebookClient is a stubbed out version of facebook.Client where an event
// in Slovenia is returned regardless of the event id requested.
type stubFacebookClient struct {
	StubError error
}

func (s stubFacebookClient) GetEventInfo(ctx context.Context, ids []string) ([]json.RawMessage, error) {
	events := make([]json.RawMessage, len(ids))
	for i, id := range ids {
		events[i] = stubEvent(id)
	}
	return events, s.StubError
}

func stubEvent(id string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(stubEventTmpl, id))
}

const stubEventTmpl = `{
	"attending_count": 8,
	"can_guests_invite": true,
	"can_viewer_post": true,
	"cover": {
		"offset_x": 0,
		"offset_y": 14,
		"source": "https://scontent.xx.fbcdn.net/v/t1.0-9/p720x720/20638182_1239062959554198_6379342660315410168_n.jpg?oh=cfc57fd95eab65664c198fddba65f48c&oe=5A1D1392",
		"id": "1239062959554198"
	},
	"declined_count": 0,
	"description": "Description",
	"end_time": "2017-08-17T20:00:00+0200",
	"guest_list_enabled": true,
	"interested_count": 36,
	"is_canceled": false,
	"is_draft": false,
	"is_page_owned": true,
	"is_viewer_admin": false,
	"id": "%s",
	"maybe_count": 36,
	"name": "VEČER ZA DUŠO",
	"noreply_count": 443,
	"owner": {
		"name": "Hiša Narave",
		"id": "356511867809316"
	},
	"place": {
		"name": "StaroMestna Čajnica Josipina",
		"location": {
		"city": "Krsko",
		"country": "Slovenia",
		"latitude": 45.962815043539,
		"longitude": 15.485937595367,
		"street": "Cesta Krških Žrtev 53",
		"zip": "8270"
		},
		"id": "1199667026764073"
	},
	"start_time": "2017-08-17T17:00:00+0200",
	"timezone": "Europe/Belgrade",
	"type": "public",
	"updated_time": "2017-08-03T10:44:57+0000"
}`

// eventGetterFunc makes an event fetching function into a facebook.Client. It
// can be used to stub out facebook.Client's GetEventInfo function.
type eventGetterFunc func(context.Context, []string) ([]json.RawMessage, error)

func (f eventGetterFunc) GetEventInfo(ctx context.Context, ids []string) ([]json.RawMessage, error) {
	return f(ctx, ids)
}

// StubTime mocks out the time with a fixed time.
type stubTime time.Time

func (s stubTime) Now() time.Time {
	return time.Time(s)
}

// StubAuth is a fake auth.Provider that takes the Authorization header and
// sets it as the current user's id. If the header equals "admin", it also sets
// the IsAdmin flag.
//
// When this auth provider is in use you can pass the JWT "user" to a rest.Client
// to simulate a user accessing the API or pass "admin" to simulate an admin
// accessing the API.
type stubAuth struct{}

func (s stubAuth) FromRequest(r *http.Request) (auth.Info, error) {
	var info auth.Info

	header := r.Header.Get("Authorization")
	if header == "" {
		return info, nil
	}

	authParts := strings.Split(header, " ")
	if len(authParts) != 2 {
		return info, errors.New("malformed Authorization header")
	}

	userID := authParts[1]

	return auth.Info{
		ID:      userID,
		IsAdmin: userID == "admin",
	}, nil
}
