package rest

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/findrandomevents/eventdb"
	"github.com/findrandomevents/eventdb/errors"
	"github.com/findrandomevents/eventdb/prom"
	"github.com/findrandomevents/eventdb/service"
)

// DestsHandler provies a REST interface to eventdb's dest-related functions.
type DestsHandler struct {
	http.Handler // router

	service *service.Service
}

func newDestsHandler(service *service.Service) *DestsHandler {
	h := &DestsHandler{
		service: service,
	}

	m := mux.NewRouter()
	m.Handle(
		"/",
		prom.InstrumentHandler("DestList", http.HandlerFunc(h.HandleList)),
	).Methods("GET")
	m.Handle(
		"/generate",
		prom.InstrumentHandler("DestGenerate", http.HandlerFunc(h.HandleGenerate)),
	).Methods("POST")
	m.Handle(
		"/{id}",
		prom.InstrumentHandler("DestGenerate", http.HandlerFunc(h.HandleGet)),
	).Methods("GET")
	m.Handle(
		"/{id}",
		prom.InstrumentHandler("DestUpdate", http.HandlerFunc(h.HandleUpdate)),
	).Methods("PATCH")
	h.Handler = m

	return h
}

// HandleList wraps Service.DestList in a REST interface
func (h *DestsHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	handleJSON(w, r, func(ctx context.Context) (interface{}, error) {
		page, _ := strconv.Atoi(r.FormValue("p"))
		return h.service.DestList(ctx, eventdb.DestListRequest{
			Page: page,
		})
	})
}

// HandleGet wraps Service.DestGet in a REST interface
func (h *DestsHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	destID := strings.TrimLeft(r.URL.Path, "/")

	handleJSON(w, r, func(ctx context.Context) (interface{}, error) {
		return h.service.DestGet(ctx, eventdb.DestID(destID))
	})
}

// HandleUpdate wraps Service.DestUpdate in a REST interface
func (h *DestsHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	destID := strings.TrimLeft(r.URL.Path, "/")
	handleJSON(w, r, func(ctx context.Context) (interface{}, error) {
		var update eventdb.DestUpdate
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			return nil, errors.E(errors.Invalid, err)
		}

		return h.service.DestUpdate(ctx, eventdb.DestID(destID), update)
	})
}

func parseGenerateRequest(r *http.Request) (eventdb.DestGenerateRequest, error) {
	var req eventdb.DestGenerateRequest

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return req, err
	}

	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			return req, err
		}
	} else {
		latStr := r.FormValue("lat")
		lat, _ := strconv.ParseFloat(latStr, 64)
		req.Lat = lat

		lngStr := r.FormValue("lng")
		lng, _ := strconv.ParseFloat(lngStr, 64)
		req.Lng = lng
	}

	userIDStr, _ := mux.Vars(r)["id"]
	req.UserID = eventdb.UserID(userIDStr)

	return req, nil
}

// HandleGenerate wraps Service.DestGenerate in a REST interface
func (h *DestsHandler) HandleGenerate(w http.ResponseWriter, r *http.Request) {
	const op errors.Op = "HandleGenerate"

	handleJSON(w, r, func(ctx context.Context) (interface{}, error) {
		req, err := parseGenerateRequest(r)
		if err != nil {
			return nil, err
		}

		return h.service.DestGenerate(ctx, req)
	})
}
