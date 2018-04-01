package service

import (
	"context"
	"math/rand"
	"time"

	"github.com/findrandomevents/eventdb"
	"github.com/findrandomevents/eventdb/auth"
	"github.com/findrandomevents/eventdb/errors"
	"github.com/findrandomevents/eventdb/geojson"
	"github.com/findrandomevents/eventdb/log"
	"go.uber.org/zap"
)

// DestGenerate finds a new random event near the user's location and returns
// a DestGenerateReply that includes the new event and whether or not the search
// was successful.
func (s *Service) DestGenerate(ctx context.Context, opts eventdb.DestGenerateRequest) (eventdb.DestGenerateReply, error) {
	const op errors.Op = "Service.DestGenerate"

	reply := eventdb.DestGenerateReply{
		Result: eventdb.GenerateOK,
		Dests:  []eventdb.Dest{},
		Events: []eventdb.Event{},
	}

	userID := opts.UserID

	currentUser := auth.User(ctx)
	if currentUser.ID == "" {
		return reply, errors.E(op, errors.Permission)
	}
	if userID == "me" || userID == "" {
		userID = eventdb.UserID(currentUser.ID)
	}
	if userID != eventdb.UserID(currentUser.ID) && !currentUser.IsAdmin { // Only admins can look up other users
		return reply, errors.E(op, errors.Permission)
	}

	chosenID, result, err := s.nextEvent(ctx, userID, opts)
	if err != nil {
		return reply, errors.E(op, errors.Internal, "nextEvent failed", err)
	}
	reply.Result = result

	if result == eventdb.GenerateOK {
		_, err = s.DestStore.Create(ctx, eventdb.Dest{
			UserID:  userID,
			EventID: chosenID,
		})
		if err != nil {
			return reply, errors.E(op, userID, errors.Internal, "create dest", err)
		}
	}

	dests, err := s.DestList(ctx, eventdb.DestListRequest{})
	if err != nil {
		return reply, errors.E(op, userID, errors.Internal, "list dests", err)
	}
	reply.Dests = dests

	destEvents := []eventdb.Event{}
	for i := range dests {
		dest := &dests[i]

		destEvents = append(destEvents, *dest.Event)
		dest.Event = nil
	}
	reply.Events = destEvents

	return reply, nil
}

// TODO(maxhawkins): clean this up

func (s *Service) nextEvent(ctx context.Context, userID eventdb.UserID, opts eventdb.DestGenerateRequest) (eventdb.EventID, eventdb.DestGenerateResult, error) {
	const op errors.Op = "Service.nextEvent"

	var chosenID eventdb.EventID

	now := time.Now()
	if s.Time != nil {
		now = s.Time.Now()
	}

	// We batch in 90 minute chunks. If the event isn't within 90m
	// we look within 180m and so on
	const timeWindow = 90 * time.Minute

	userLat, userLng := opts.Lat, opts.Lng

	// ~5mi radius
	const radiusM = 8000.0
	bounds := geojson.CircleGeom(userLat, userLng, radiusM)

	// Get a list of existing dests so we don't repeat
	alreadyChosen, err := s.DestStore.ListForUser(ctx, userID, eventdb.DestListRequest{})
	if err != nil {
		return chosenID, eventdb.GenerateError, errors.E(op, userID, err, "list dests")
	}

	if len(alreadyChosen) > 0 {
		lastDest := alreadyChosen[0]
		lastEvent, err := s.EventStore.GetByID(ctx, lastDest.EventID)
		if err != nil {
			return chosenID, eventdb.GenerateError, errors.E(op, userID, err, "get last event")
		}

		if lastEvent.StartTime.After(now) {
			return chosenID, eventdb.GenerateWait, nil
		}
	}

	// Start searching 10m out (allow for travel time)
	searchTime := now.Add(10 * time.Minute)

	// TODO(maxhawkins): if it's your first event or you haven't been to one in a while,
	// favor events that are really close by. It's easier to get going.

	for {
		// If there's nothing in the next two days we don't have anything in the db
		if searchTime.Sub(now) > 48*time.Hour {
			return chosenID, eventdb.GenerateNoResults, nil
		}

		events, err := s.EventStore.Search(ctx, eventdb.EventSearchRequest{
			Bounds: bounds,
			Start:  searchTime,
			End:    searchTime.Add(timeWindow),
		})
		if errors.Is(errors.NotExist, err) {
			return chosenID, eventdb.GenerateNoResults, nil
		}
		if err != nil {
			return chosenID, eventdb.GenerateError, errors.E(op, userID, "search failed", err)
		}

		var goodEvents []eventdb.Event
		for _, event := range events {
			var badEvent bool

			// Filter out things we've already suggested
			for _, chosen := range alreadyChosen {
				if chosen.EventID == event.ID {
					badEvent = true
					break
				}
			}

			// TODO(maxhawkins): if it's far away, make this longer
			// As a rule of thumb, if it takes longer to get there than you'll
			// be able to spend at the event it should be filteredq

			// Filter out things that will end when we arrive
			arriveTime := now.Add(30 * time.Minute)
			if arriveTime.After(event.EndTime) {
				badEvent = true
			}

			// The good ones get added to the list
			if !badEvent {
				goodEvents = append(goodEvents, event)
			}
		}

		// If there aren't any candidates, look 90m further into the future
		if len(goodEvents) == 0 {
			searchTime = searchTime.Add(timeWindow)
			continue
		}

		// Now find a random event
		n := rand.Intn(len(goodEvents))
		return goodEvents[n].ID, eventdb.GenerateOK, nil
	}
}

// DestUpdate updates a Dest with the user's feedback
func (s *Service) DestUpdate(ctx context.Context, id eventdb.DestID, update eventdb.DestUpdate) (eventdb.Dest, error) {
	const op errors.Op = "Service.DestUpdate"

	dest, err := s.DestStore.Get(ctx, id)
	if err != nil {
		return dest, err
	}

	currentUser := auth.User(ctx)
	if !currentUser.IsAdmin && currentUser.ID != string(dest.UserID) {
		return dest, errors.E(op, errors.Permission, currentUser.ID)
	}

	dest, err = s.DestStore.Update(ctx, id, update)
	if err != nil {
		return dest, errors.E(op, currentUser.ID, err)
	}

	return dest, nil
}

// DestGet retrieves a Dest from the database.
func (s *Service) DestGet(ctx context.Context, id eventdb.DestID) (eventdb.Dest, error) {
	const op errors.Op = "Service.DestGet"

	logger := log.FromContext(ctx)

	currentUser := auth.User(ctx)

	dest, err := s.DestStore.Get(ctx, id)
	if err != nil {
		return dest, errors.E(op, currentUser.ID, err)
	}

	if !currentUser.IsAdmin && currentUser.ID != string(dest.UserID) {
		return dest, errors.E(op, errors.Permission, currentUser.ID)
	}

	event, err := s.EventStore.GetByID(ctx, dest.EventID)
	if err == nil {
		dest.Event = &event
	} else {
		logger.Error("failed to get event",
			zap.Error(err),
			zap.String("eventID", string(dest.EventID)))
	}

	return dest, nil
}

// DestList lists a user's Dests by creation date.
func (s *Service) DestList(ctx context.Context, opts eventdb.DestListRequest) ([]eventdb.Dest, error) {
	const op errors.Op = "Service.DestList"

	userID := auth.User(ctx).ID
	if userID == "" {
		return nil, errors.E(op, errors.NotLoggedIn)
	}

	dests, err := s.DestStore.ListForUser(ctx, eventdb.UserID(userID), opts)
	if err != nil {
		return nil, errors.E(op, userID, err)
	}

	// Side-load the events
	var eventIDs []eventdb.EventID
	for _, dest := range dests {
		eventIDs = append(eventIDs, dest.EventID)
	}
	events, err := s.EventStore.GetMulti(ctx, eventIDs)
	if err != nil {
		return nil, errors.E(op, userID, err)
	}

	// TODO(maxhawkins): optimize with a join
	for i := range dests {
		dest := &dests[i]

		for _, event := range events {
			if dest.EventID == event.ID {
				dest.Event = &event
				break
			}
		}
	}

	return dests, nil
}
