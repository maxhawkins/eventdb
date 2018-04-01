package service

import (
	"context"

	"github.com/findrandomevents/eventdb"
	"github.com/findrandomevents/eventdb/auth"
	"github.com/findrandomevents/eventdb/errors"
)

// UserUpdate lets users update their profile data.
func (s *Service) UserUpdate(ctx context.Context, id eventdb.UserID, update eventdb.UserUpdate) (*eventdb.User, error) {
	const op errors.Op = "Service.UserUpdate"

	currentUser := auth.User(ctx)
	if id != "me" {
		return nil, errors.E(op, errors.Permission, currentUser.ID)
	}
	id = eventdb.UserID(currentUser.ID)

	updatedUser, err := s.UserStore.Update(ctx, id, update)
	if err != nil {
		return nil, errors.E(op, errors.Permission, currentUser.ID, err)
	}

	return &updatedUser, nil
}

// UserGet retrieves User records.
func (s *Service) UserGet(ctx context.Context, id eventdb.UserID) (eventdb.User, error) {
	const op errors.Op = "Service.UserGet"

	var user eventdb.User

	currentUser := auth.User(ctx)
	if id != "me" {
		return user, errors.E(op, errors.Permission, currentUser.ID)
	}
	id = eventdb.UserID(currentUser.ID)

	user, err := s.UserStore.GetByID(ctx, id)
	if err != nil {
		return user, errors.E(op, errors.Internal, currentUser.ID, err)
	}

	return user, nil
}
