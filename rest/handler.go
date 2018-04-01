// Package rest contains a REST handler for eventdb. It wraps Service in a
// web-accessible API.
package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"path"
	"strings"

	"go.uber.org/zap"

	"github.com/findrandomevents/eventdb/auth"
	"github.com/findrandomevents/eventdb/errors"
	"github.com/findrandomevents/eventdb/log"
	"github.com/findrandomevents/eventdb/service"
)

// New creates a new REST service wrapping an eventdb Service.
func New(service *service.Service) *Handler {
	return &Handler{
		Auth: service.Auth,

		UsersHandler:  newUsersHandler(service),
		EventsHandler: newEventsHandler(service),
		DestsHandler:  newDestsHandler(service),
	}
}

// Handler is an http.Handler that provides a REST interface for eventdb.
type Handler struct {
	Auth auth.Provider

	UsersHandler  *UsersHandler
	EventsHandler *EventsHandler
	DestsHandler  *DestsHandler
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var head string
	head, r.URL.Path = ShiftPath(r.URL.Path)

	// Retrieve the logger from HTTP middleware, if set.
	ctx := r.Context()
	logger := log.FromContext(ctx)

	// Get auth info from the JWT header
	user, err := h.Auth.FromRequest(r)
	if err == auth.ErrExpired {
		writeErrorResp(w, errors.Response{
			Error:  "auth token expired",
			Status: http.StatusUnauthorized,
		})
		return

	} else if err != nil {
		logger.Warn("parse auth failed", zap.Error(err))
	}
	ctx = user.WithContext(ctx)

	// Decorate the logger with the user id
	logger = logger.With(zap.String("userid", user.ID))
	ctx = log.ToContext(ctx, logger)
	r = r.WithContext(ctx)

	switch head {
	case "users":
		if h.UsersHandler != nil {
			h.UsersHandler.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}

	case "events":
		if h.EventsHandler != nil {
			h.EventsHandler.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}

	case "dests":
		if h.DestsHandler != nil {
			h.DestsHandler.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}

	case "healthz":
		if rand.Intn(2) == 0 {
			fmt.Fprintln(w, "heads")
		} else {
			fmt.Fprintln(w, "tails")
		}

	case "":
		http.Redirect(w, r, "https://findrandomevents.com", http.StatusTemporaryRedirect)

	default:
		http.NotFound(w, r)
	}
}

// ShiftPath splits off the first component of p, which will be cleaned of
// relative components before processing. head will never contain a slash and
// tail will always be a rooted path without trailing slash.
func ShiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], "/"
	}
	return p[1:i], p[i:]
}

func handleJSON(w http.ResponseWriter, r *http.Request, f func(context.Context) (interface{}, error)) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	resp, err := f(ctx)
	if err != nil {
		errResp := errors.ResponseForError(err)
		if errResp.Status >= 500 {
			logger.Error("internal server error", zap.Error(err))
		} else {
			logger.Warn("handler failed", zap.Error(err))
		}

		if auth.User(ctx).IsAdmin { // show the full error if it's an admin
			errResp.Error = fmt.Sprintf("%s: %s", errResp.Error, err.Error())
		}

		writeErrorResp(w, errResp)
		return
	}

	js, err := json.MarshalIndent(resp, "", "\t")
	if err != nil {
		logger.Error("write json failed", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(js)
}

func writeErrorResp(w http.ResponseWriter, resp errors.Response) {
	js, err := json.MarshalIndent(resp, "", "\t")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.Status)
	w.Write(js)
}
