package eventdb

import (
	"time"
)

// UserID is used to identify Users. Right now it's a Firebase UID.
type UserID string

// User stores metadata about a Third Party user
type User struct {
	ID            UserID    `json:"id"`
	TimeZone      string    `json:"timeZone"`
	FacebookID    string    `json:"facebookID"`
	FacebookToken string    `json:"facebookToken"`
	Birthday      time.Time `json:"birthday"`
}

// A UserUpdate is used to update a User object
type UserUpdate struct {
	TimeZone      string    `json:"timeZone"`
	FacebookID    string    `json:"facebookID"`
	FacebookToken string    `json:"facebookToken"`
	Birthday      time.Time `json:"birthday"`
	// Mask is a comma-delimited list of json names for the fields this update
	// will change. Only fields listed in the mask will be updated.
	//
	// eg: "timeZone,birthday" means this update changes TimeZone and Birthday
	//
	// This is similar to protobuf's FieldMask well known type.
	Mask string `json:"mask"`
}
