package errors

import (
	"context"
	"net/http"
)

// Response is a JSON-serializable version of an Error. It can be used to
// transmit errors across the REST API.
type Response struct {
	Error   string      `json:"error,omitempty"`
	Details interface{} `json:"details,omitempty"`
	Status  int         `json:"status,omitempty"`
}

// ToError converts an ErrorResponse back into an Error
func (e Response) ToError() error {
	switch e.Status {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return E(NotLoggedIn, e.Error)
	case http.StatusForbidden:
		return E(Permission, e.Error)
	case http.StatusBadRequest:
		return E(Invalid, e.Error)
	case http.StatusConflict:
		return E(Exist, e.Error)
	case http.StatusNotFound:
		return E(NotExist, e.Error)
	}
	return Errorf("status %d: %s", e.Status, e.Error)
}

// ResponseForError constructs an ErrorResponse based on an Error. Since this
// object is user-visible it's not a 1-1 mapping. Some errors will return
// detailed information about why the error happened in the Error and Details
// sections. Others wil just return an opaque error type.
func ResponseForError(err error) Response {
	return Response{
		Error:   errText(err),
		Details: errDetails(err),
		Status:  errStatus(err),
	}
}

func errText(err error) string {
	if e, ok := err.(*Error); ok {
		switch e.Kind {
		case Permission:
			return "access to this endpoint is restricted. contact max@findrandomevents.com for more information."
		case NotLoggedIn:
			return "not logged in: please authenticate with firebase and send the token as an Authorization header"
		case Invalid:
			return e.Error()
		}
	}

	return http.StatusText(errStatus(err))
}

func errDetails(err error) interface{} {
	return nil
}

func errStatus(err error) int {
	switch err {
	case context.Canceled:
		return http.StatusBadRequest
	}

	if e, ok := err.(*Error); ok {
		switch e.Kind {
		case Other:
			return http.StatusInternalServerError
		case Invalid:
			return http.StatusBadRequest
		case NotLoggedIn:
			return http.StatusUnauthorized
		case Permission:
			return http.StatusForbidden
		case NotExist:
			return http.StatusNotFound
		case Exist:
			return http.StatusConflict
		case Internal:
			return http.StatusInternalServerError
		default:
			return http.StatusInternalServerError
		}
	}

	return http.StatusInternalServerError
}
