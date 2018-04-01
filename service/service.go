package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/findrandomevents/eventdb/auth"
	"github.com/findrandomevents/eventdb/pg"
)

// Time mocks out time.Now for testing
type Time interface {
	Now() time.Time
}

// Service is a programmatic API to the eventdb. It manages access to the Store
// and checks permissions.
type Service struct {
	DestStore  *pg.DestStore
	EventStore *pg.EventStore
	UserStore  *pg.UserStore

	FacebookClient func(oauthToken string) FacebookClient
	Time           Time

	Auth auth.Provider
}

// FacebookClient mocks out access to the Facebook Graph API.
type FacebookClient interface {
	GetEventInfo(ctx context.Context, ids []string) ([]json.RawMessage, error)
}
