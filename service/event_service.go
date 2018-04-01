package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/findrandomevents/eventdb"
	"github.com/findrandomevents/eventdb/auth"
	"github.com/findrandomevents/eventdb/errors"
	"github.com/findrandomevents/eventdb/facebook"
)

// EventSearch queries the database for events matching the EventSearchRequest
// and returns Event objects for the matching results.
func (s *Service) EventSearch(ctx context.Context, req eventdb.EventSearchRequest) ([]eventdb.Event, error) {
	const op errors.Op = "Service.EventSearch"

	if !auth.User(ctx).IsAdmin {
		return nil, errors.E(op, errors.Permission)
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	events, err := s.EventStore.Search(ctx, req)
	if err != nil {
		err = errors.E(op, errors.Internal, "event search", err)
		return nil, err
	}

	for i := range events {
		desc := events[i].Description
		if len(desc) > 100 {
			events[i].Description = desc[:97] + "…"
		}
	}

	return events, nil
}

// EventSearchFull queries the database for events matching the EventSearchRequest
// and returns the raw Graph API JSON data for the matching results.
func (s *Service) EventSearchFull(ctx context.Context, params eventdb.EventSearchRequest) ([]json.RawMessage, error) {
	const op errors.Op = "Service.EventSearchFull"

	if !auth.User(ctx).IsAdmin {
		return nil, errors.E(op, errors.Permission)
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	return s.EventStore.SearchFull(ctx, params)
}

// EventGet retrieves an event from the database.
func (s *Service) EventGet(ctx context.Context, id eventdb.EventID) (eventdb.Event, error) {
	const op errors.Op = "Service.EventGet"

	event, err := s.EventStore.GetByID(ctx, id)
	if err != nil {
		return event, errors.E(op, errors.Internal, "event get failed", err)
	}

	return event, err
}

// EventSubmit downloads the events using the Facebook API and saves them to the
// EventStore. It uses a random user's Facebook API token to fetch the event
// so some users must be logged in with Facebook for this method to work.
func (s *Service) EventSubmit(ctx context.Context, req eventdb.EventSubmitRequest) error {
	const op errors.Op = "Service.EventSubmit"

	userID := eventdb.UserID(auth.User(ctx).ID)

	if userID == "" {
		return errors.E(op, errors.Permission)
	}

	eventIDs := req.EventIDs
	if len(eventIDs) > 50 {
		err := fmt.Errorf("event list length (%d) > max (50)", len(eventIDs))
		return errors.E(op, errors.Invalid, userID, err)
	}

	err := retry(ctx, 3, func() error {
		fetcherID, oauthToken, err := s.UserStore.RandomFBToken(ctx)
		if err != nil {
			return errors.E(op, errors.Internal, userID, err)
		}

		client := s.FacebookClient(oauthToken)

		var eventIDStrs []string
		for _, id := range eventIDs {
			eventIDStrs = append(eventIDStrs, string(id))
		}

		events, err := client.GetEventInfo(ctx, eventIDStrs)
		if facebook.IsTokenExpired(err) {
			_, err = s.UserStore.Update(ctx, fetcherID, eventdb.UserUpdate{
				FacebookToken: "",
				Mask:          "facebookToken",
			})
			if err != nil {
				return errors.E(op, userID, "expire user token", err)
			}
			return errors.E(op, userID, "facebook token expired")

		} else if err != nil {
			return err
		}

		for _, e := range events {
			event, err := s.EventStore.Save(ctx, e)
			if err != nil {
				return errors.E(op, errors.Internal, "save event", err)
			}

			if err := s.EventStore.SetBad(ctx, event.ID, eventdb.IsBadEvent(event)); err != nil {
				return errors.E(op, errors.Internal, "mark bad", err)
			}
		}

		return nil
	})
	if err != nil {
		return errors.E(op, err)
	}

	return nil
}

// retry is a simple exponential backoff function. If you cancel the context
// passed to it retries will stop.
func retry(ctx context.Context, count int, f func() error) error {
	retries := count

RETRY:
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := f(); err != nil {
		if retries == 0 {
			return err
		}

		retries--
		backoff := (math.Pow(2, float64(retries)) + rand.Float64()) * float64(time.Second)
		time.Sleep(time.Duration(backoff))
		goto RETRY
	}

	return nil
}
