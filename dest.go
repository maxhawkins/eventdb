package eventdb

import (
	"time"
)

// DestID is an identifier for a Dest.
type DestID string

// Dest records a User's destination: a random event selected for them to attend.
//
// It can be updated with feedback from the user and information about whether
// they went or not.
type Dest struct {
	ID      DestID  `json:"id"`
	UserID  UserID  `json:"userID"`
	EventID EventID `json:"eventID"`

	// Used to side-load event data when sending dest list to the client
	Event *Event `json:"event,omitempty"`

	Status   string `json:"status"`
	Feedback string `json:"feedback"`

	CreatedAt time.Time `json:"createdAt"`
}

// A DestUpdate allows a user to update a Dest with feedback.
type DestUpdate struct {
	Feedback string `json:"feedback"`
	Status   string `json:"status"`
	// Mask is a comma-delimited list of json names for the fields this update
	// will change. Only fields listed in the mask will be updated.
	//
	// eg: "feedback" means this update changes only Feedback.
	//
	// This is similar to protobuf's FieldMask well known type.
	Mask string `json:"mask"`
}

// DestGenerateRequest is a request for a Dest at a given location.
//
// It's sent by the client to get their next random event.
type DestGenerateRequest struct {
	UserID UserID  `json:"userID"`
	Lat    float64 `json:"lat"`
	Lng    float64 `json:"lng"`
}

// DestGenerateResult describes whether or not a DestGenerate request was
// fulfilled, and if not why.
type DestGenerateResult string

const (
	// GenerateOK means a destination was generated successfully.
	GenerateOK DestGenerateResult = "ok"
	// GenerateWait means the user needs to wait a while before requesting a new
	// destination, and no destination was generated.
	GenerateWait DestGenerateResult = "wait"
	// GenerateNoResults means that no upcoming events were found in the requested
	// area. Try again later or in another place.
	GenerateNoResults DestGenerateResult = "no-results"
	// GenerateError means there was a problem generating the event, try again later
	GenerateError DestGenerateResult = "error"
)

// DestGenerateReply is returned in response to a DestGenerateRequest. It reports
// whether a new destination was generated, and lists a few of the most recently
// generated destinations.
type DestGenerateReply struct {
	Result DestGenerateResult `json:"result"`
	Dests  []Dest             `json:"dests"`
	Events []Event            `json:"events"`
}

// A DestListRequest requests a piece of the user's dest list.
type DestListRequest struct {
	Page int `json:"page"`
}
