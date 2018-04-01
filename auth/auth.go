package auth

import (
	"context"
	"errors"
	"net/http"
)

// ErrExpired is returned when the user tries to autneticate with an expired token.
var ErrExpired = errors.New("token expired")

// Provider parses requests to extract authorization info.
type Provider interface {
	FromRequest(r *http.Request) (Info, error)
}

// Info stores information about the current user
type Info struct {
	ID      string
	IsAdmin bool
}

// WithContext decorates a context with this auth.Info object. Use auth.User
// to retrieve the auth.Info from the context.
func (i Info) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxMarkerKey, i)
}
