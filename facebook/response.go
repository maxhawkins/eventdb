package facebook

import (
	"encoding/json"
	"fmt"
	"io"
)

// Error is an error returned by the Facebook Graph API
type Error struct {
	Message   string `json:"message"`
	Type      string `json:"type"`
	Code      int    `json:"code"`
	Subcode   int    `json:"error_subcode"`
	FBTraceID string `json:"fbtrace_id"`
}

func (f Error) Error() string {
	return fmt.Sprintf("%s type=%q code=%d subcode=%d", f.Message, f.Type, f.Code, f.Subcode)
}

// ErrorResponse contains an Error. It's returned by the Graph API.
type ErrorResponse struct {
	Error Error `json:"error"`
}

func parseError(body io.Reader) Error {
	var resp ErrorResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		msg := fmt.Sprintf("failed to decode error: %v", err)
		return Error{Message: msg}
	}

	return resp.Error
}

// BatchResponse contains information about one operation in a Graph API batch
// request.
type BatchResponse struct {
	Code int    `json:"code"`
	Body string `json:"body"`
}

// IsTokenExpired returns true if this is a token expired error from the
// Facebook API client.
func IsTokenExpired(err error) bool {
	e, ok := err.(Error)
	if !ok {
		return false
	}
	return e.Type == "OAuthException" && e.Code == 190
}
