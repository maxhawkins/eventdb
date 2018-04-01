package rest

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/findrandomevents/eventdb"
	"github.com/findrandomevents/eventdb/errors"
	"github.com/findrandomevents/eventdb/prom"
	"github.com/findrandomevents/eventdb/service"
)

// EventsHandler provies a REST interface to eventdb's event-related functions.
type EventsHandler struct {
	http.Handler // router

	service *service.Service
}

func newEventsHandler(service *service.Service) *EventsHandler {
	h := &EventsHandler{
		service: service,
	}

	m := mux.NewRouter()
	m.Handle(
		"/",
		prom.InstrumentHandler("EventSubmit", http.HandlerFunc(h.HandleSubmit)),
	).Methods("POST")
	m.Handle(
		"/search",
		prom.InstrumentHandler("EventSearch", http.HandlerFunc(h.HandleSearch)),
	).Methods("POST", "GET")
	m.Handle(
		"/{id}",
		prom.InstrumentHandler("EventGet", http.HandlerFunc(h.HandleGet)),
	).Methods("GET")

	h.Handler = m

	return h
}

// HandleGet wraps Service.EventGet in a REST interface
func (h *EventsHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	eventID, _ := mux.Vars(r)["id"]

	handleJSON(w, r, func(ctx context.Context) (interface{}, error) {
		return h.service.EventGet(ctx, eventdb.EventID(eventID))
	})
}

// HandleSubmit wraps Service.EventSubmit in a REST interface
func (h *EventsHandler) HandleSubmit(w http.ResponseWriter, r *http.Request) {
	handleJSON(w, r, func(ctx context.Context) (interface{}, error) {
		var req eventdb.EventSubmitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, errors.E(errors.Invalid, err)
		}

		if err := h.service.EventSubmit(ctx, req); err != nil {
			return nil, err
		}
		return nil, nil
	})
}

// HandleSearch wraps Service.EventSearch in a REST interface
func (h *EventsHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	handleJSON(w, r, func(ctx context.Context) (interface{}, error) {
		var js []byte
		var err error

		if r.FormValue("json") != "" {
			js = []byte(r.FormValue("json"))
		} else {
			js, err = ioutil.ReadAll(r.Body)
			if err != nil {
				return nil, errors.E(errors.Invalid, err)
			}
		}

		var params eventdb.EventSearchRequest
		if err := json.Unmarshal(js, &params); err != nil {
			return nil, errors.E(errors.Invalid, err)
		}

		if r.FormValue("format") == "full" {
			return h.service.EventSearchFull(ctx, params)
		}
		return h.service.EventSearch(ctx, params)
	})
}
