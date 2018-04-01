package rest

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/findrandomevents/eventdb"
	"github.com/findrandomevents/eventdb/errors"
	"github.com/findrandomevents/eventdb/prom"
	"github.com/findrandomevents/eventdb/service"

	"github.com/gorilla/mux"
)

// UsersHandler provides a REST interface to eventdb's user-related functions.
type UsersHandler struct {
	http.Handler // router

	service *service.Service
}

func newUsersHandler(service *service.Service) *UsersHandler {
	h := &UsersHandler{
		service: service,
	}

	m := mux.NewRouter()
	m.Handle(
		"/{id}",
		prom.InstrumentHandler("UserGet", http.HandlerFunc(h.HandleGet)),
	).Methods("GET")
	m.Handle(
		"/{id}",
		prom.InstrumentHandler("UserUpdate", http.HandlerFunc(h.HandleUpdate)),
	).Methods("PATCH")
	h.Handler = m

	return h
}

// HandleUpdate wraps Service.UserUpdate in a REST interface
func (h *UsersHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	userID, _ := mux.Vars(r)["id"]

	handleJSON(w, r, func(ctx context.Context) (interface{}, error) {
		var update eventdb.UserUpdate
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			return nil, errors.E(errors.Invalid, err)
		}

		updated, err := h.service.UserUpdate(ctx, eventdb.UserID(userID), update)
		if err != nil {
			return nil, err
		}

		return updated, nil
	})
}

// HandleGet wraps Service.UserGet in a REST interface
func (h *UsersHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	userID, _ := mux.Vars(r)["id"]

	handleJSON(w, r, func(ctx context.Context) (interface{}, error) {
		user, err := h.service.UserGet(ctx, eventdb.UserID(userID))
		if err != nil {
			return nil, errors.E(err, "get user")
		}

		return user, nil
	})
}
