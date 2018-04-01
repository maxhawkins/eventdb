package eventdb

import (
	"time"
)

// EventID is a string assigned by Facebook that uniquely identifies the Event.
// You can access the event it references at https://facebook.com/<event id>.
type EventID string

// Event describes a (random) Facebook event.
type Event struct {
	// These fields are extracted from the Facebook Graph API response
	ID          EventID   `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	IsCanceled  bool      `json:"is_canceled"`
	Cover       string    `json:"cover"`
	Place       string    `json:"place"`
	Address     string    `json:"address"`

	// IsBad is a flag used to filter events that don't work well on the service.
	//
	// But what is bad, really? I'm thinking about removing this field and
	// replacing it with something more thoroughly thought out. See the discussion
	// at IsBadEvent().
	IsBad bool `json:"is_bad"`
}

// EventSearchRequest is passed to EventStore.Search to find events at a certain time
// and place.
type EventSearchRequest struct {
	Bounds     string    `json:"bounds"`
	Start      time.Time `json:"start"`
	End        time.Time `json:"end"`
	IncludeBad bool      `json:"includeBad"`
}

// An EventSubmitRequest is a request to add a facebook event to the event database.
type EventSubmitRequest struct {
	// EventIDs are the Facebook Event IDs.
	//
	// Submissions can be batched for efficiency. Up to 50 ids may be submitted at a time.
	EventIDs []EventID `json:"event_ids"`
}
